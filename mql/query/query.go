// Package query provides a typed builder for MongoDB query filters,
// intended for use as the argument to agg.MatchStage.
//
// Only a starter set of field conditions is implemented; this package is
// designed to grow independently of the aggregation expression system.
package query

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FieldCondition represents a condition applied to a single document field,
// e.g. { $gt: value }. Construct via the operator functions (Eq, Gt, etc.).
type FieldCondition struct{ doc bson.D }

// Filter represents a complete MongoDB query document, e.g. { field: { $gt: v } }.
// Construct via Field or the logical combinators And and Or.
type Filter bson.D

// Field creates a Filter for the named field from one or more FieldConditions
// (constructed via Eq, Gt, etc.). Multiple conditions are merged into a single
// document to apply several conditions to the same field, e.g.
//
//	query.Field("qty", query.Gt(5), query.Lt(15))
//	// { qty: { $gt: 5, $lt: 15 } }
func Field(name string, conds ...FieldCondition) Filter {
	merged := bson.D{}
	for _, c := range conds {
		merged = append(merged, c.doc...)
	}
	return Filter{{Key: name, Value: merged}}
}

// Option is a functional option that configures the optional parameters of an
// operator with type T.
type Option[T any] func(*T)

// Number is the set of Go numeric types accepted where the MQL spec calls for
// a number.
type Number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

// All creates a FieldCondition matching arrays that contain all of the given
// values: { $all: [ ... ] }. Values are usually plain scalars, but may also be
// ElemMatch conditions to match arrays of embedded documents.
func All(values ...any) FieldCondition {
	arr := make(bson.A, len(values))
	for i, v := range values {
		if fc, ok := v.(FieldCondition); ok {
			arr[i] = fc.doc
		} else {
			arr[i] = v
		}
	}
	return FieldCondition{doc: bson.D{{Key: "$all", Value: arr}}}
}

// And creates a Filter for logical AND: { $and: [ filter1, filter2, ... ] }.
func And(filters ...Filter) Filter {
	clauses := make(bson.A, 0, len(filters))
	for _, f := range filters {
		clauses = append(clauses, bson.D(f))
	}
	return Filter{{Key: "$and", Value: clauses}}
}

// Box creates a legacy rectangular box geometry ($box) from the bottom-left and
// top-right coordinate pairs, for use with GeoWithin.
func Box(bottomLeft, topRight []float64) Geometry {
	return geometry{doc: bson.D{{Key: "$box", Value: bson.A{bottomLeft, topRight}}}}
}

// Center creates a legacy circle geometry ($center) from a center coordinate
// pair and a radius (in coordinate units), for use with GeoWithin using planar
// geometry.
func Center(center []float64, radius float64) Geometry {
	return geometry{doc: bson.D{{Key: "$center", Value: bson.A{center, radius}}}}
}

// CenterSphere creates a legacy spherical circle geometry ($centerSphere) from
// a center coordinate pair and a radius (in radians), for use with GeoWithin
// using spherical geometry.
func CenterSphere(center []float64, radius float64) Geometry {
	return geometry{doc: bson.D{{Key: "$centerSphere", Value: bson.A{center, radius}}}}
}

// ElemMatch creates a FieldCondition matching arrays with at least one element
// that satisfies all of the given queries: { $elemMatch: { ... } }. Pass Filters
// to match arrays of embedded documents (e.g. query.Field("product", query.Eq("xyz")))
// or FieldConditions to match scalar elements (e.g. query.Gte(80), query.Lt(85)).
func ElemMatch[T Filter | FieldCondition](queries ...T) FieldCondition {
	inner := bson.D{}
	for _, q := range queries {
		switch v := any(q).(type) {
		case Filter:
			inner = append(inner, v...)
		case FieldCondition:
			inner = append(inner, v.doc...)
		}
	}
	return FieldCondition{doc: bson.D{{Key: "$elemMatch", Value: inner}}}
}

// Eq creates a FieldCondition for equality: { $eq: value }.
func Eq(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$eq", Value: value}}}
}

// Exists creates a FieldCondition matching documents that have (or lack) the
// field: { $exists: exists }.
func Exists(exists bool) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$exists", Value: exists}}}
}

// Geometry represents a geometry value supplied to the geospatial query
// operators (GeoWithin, GeoIntersects, Near, NearSphere). Construct via GeoJson
// or the legacy shape helpers (Box, Center, CenterSphere, Polygon).
type Geometry interface{ geometry() geometry }

type geometry struct {
	doc bson.D
}

func (g geometry) geometry() geometry { return g }

// Coordinates constrains the coordinate values of a GeoJson geometry to array
// shapes. The nesting depth depends on the geometry type: a Point is a single
// position ([]float64), a Polygon is [][][]float64, and so on. Use bson.A for
// dynamic or mixed shapes.
type Coordinates interface {
	[]float64 | [][]float64 | [][][]float64 | [][][][]float64 | bson.A
}

type geoJsonOptions struct {
	crs bson.D
}

// WithGeoJsonCrs sets the coordinate reference system for a GeoJson geometry,
// e.g. to request a big (strict CRS84) polygon.
func WithGeoJsonCrs(crs bson.D) Option[geoJsonOptions] {
	return func(o *geoJsonOptions) {
		o.crs = crs
	}
}

// GeoJson creates a GeoJson geometry ($geometry) of the given type (e.g.
// "Point", "Polygon") and coordinates. The coordinate shape depends on the
// geometry type. Optionally specify a coordinate reference system via
// WithGeoJsonCrs.
func GeoJson[C Coordinates](geoType string, coordinates C, opts ...Option[geoJsonOptions]) Geometry {
	var o geoJsonOptions
	for _, opt := range opts {
		opt(&o)
	}
	geo := bson.D{
		{Key: "type", Value: geoType},
		{Key: "coordinates", Value: coordinates},
	}
	if o.crs != nil {
		geo = append(geo, bson.E{Key: "crs", Value: o.crs})
	}
	return geometry{doc: bson.D{{Key: "$geometry", Value: geo}}}
}

// GeoIntersects creates a FieldCondition matching geometries that intersect the
// given GeoJson geometry: { $geoIntersects: { $geometry: ... } }. Use GeoJson to
// construct the geometry; $geoIntersects does not support the legacy shapes.
func GeoIntersects(g Geometry) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$geoIntersects", Value: g.geometry().doc}}}
}

// GeoWithin creates a FieldCondition matching geometries within the given
// bounding geometry: { $geoWithin: { ... } }. Accepts a GeoJson geometry or any
// of the legacy shapes (Box, Center, CenterSphere, Polygon).
func GeoWithin(g Geometry) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$geoWithin", Value: g.geometry().doc}}}
}

// Gt creates a FieldCondition for greater than: { $gt: value }.
func Gt(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$gt", Value: value}}}
}

// Gte creates a FieldCondition for greater than or equal to: { $gte: value }.
func Gte(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$gte", Value: value}}}
}

// In creates a FieldCondition matching any of the given values: { $in: [ ... ] }.
func In(values ...any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$in", Value: bson.A(values)}}}
}

// Lt creates a FieldCondition for less than: { $lt: value }.
func Lt(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$lt", Value: value}}}
}

// Lte creates a FieldCondition for less than or equal to: { $lte: value }.
func Lte(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$lte", Value: value}}}
}

// MaxDistance creates a FieldCondition limiting Near and NearSphere results to
// at most the given distance from the center point: { $maxDistance: value }.
func MaxDistance[T Number](value T) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$maxDistance", Value: value}}}
}

// MinDistance creates a FieldCondition limiting Near and NearSphere results to
// at least the given distance from the center point: { $minDistance: value }.
func MinDistance[T Number](value T) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$minDistance", Value: value}}}
}

type nearOptions struct {
	minDistance any
	maxDistance any
}

// WithNearMinDistance limits Near/NearSphere results to at least the given
// distance (in meters) from the center point.
func WithNearMinDistance[T Number](d T) Option[nearOptions] {
	return func(o *nearOptions) { o.minDistance = d }
}

// WithNearMaxDistance limits Near/NearSphere results to at most the given
// distance (in meters) from the center point.
func WithNearMaxDistance[T Number](d T) Option[nearOptions] {
	return func(o *nearOptions) { o.maxDistance = d }
}

// nearDoc merges the geometry with the optional distance bounds into the value
// document shared by $near and $nearSphere.
func nearDoc(g Geometry, opts []Option[nearOptions]) bson.D {
	var o nearOptions
	for _, opt := range opts {
		opt(&o)
	}
	doc := append(bson.D(nil), g.geometry().doc...)
	if o.minDistance != nil {
		doc = append(doc, bson.E{Key: "$minDistance", Value: o.minDistance})
	}
	if o.maxDistance != nil {
		doc = append(doc, bson.E{Key: "$maxDistance", Value: o.maxDistance})
	}
	return doc
}

// Ne creates a FieldCondition matching values not equal to value: { $ne: value }.
func Ne(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$ne", Value: value}}}
}

// Near creates a FieldCondition matching geospatial objects in proximity to the
// given point, sorted by distance: { $near: { $geometry: ..., $minDistance,
// $maxDistance } }. Bounds are set via WithNearMinDistance and
// WithNearMaxDistance.
func Near(g Geometry, opts ...Option[nearOptions]) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$near", Value: nearDoc(g, opts)}}}
}

// NearSphere creates a FieldCondition matching geospatial objects in proximity
// to the given point on a sphere, sorted by distance: { $nearSphere: {
// $geometry: ..., $minDistance, $maxDistance } }. Bounds are set via
// WithNearMinDistance and WithNearMaxDistance.
func NearSphere(g Geometry, opts ...Option[nearOptions]) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$nearSphere", Value: nearDoc(g, opts)}}}
}

// Nin creates a FieldCondition matching none of the given values: { $nin: [ ... ] }.
func Nin(values ...any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$nin", Value: bson.A(values)}}}
}

// Nor creates a Filter for logical NOR, matching documents that fail every
// clause: { $nor: [ filter1, filter2, ... ] }.
func Nor(filters ...Filter) Filter {
	clauses := make(bson.A, 0, len(filters))
	for _, f := range filters {
		clauses = append(clauses, bson.D(f))
	}
	return Filter{{Key: "$nor", Value: clauses}}
}

// Not creates a FieldCondition that inverts another field-level condition:
// { $not: <arg> }. The argument may be a FieldCondition, e.g.
// query.Not(query.Gt(1.99)) yields { $not: { $gt: 1.99 } }, or a bson.Regex,
// e.g. query.Not(bson.Regex{Pattern: "^p.*"}) yields { $not: /^p.*/ }. MongoDB's
// $not requires an operator expression or regex; it does not accept a plain
// scalar value.
func Not[T FieldCondition | bson.Regex](arg T) FieldCondition {
	var value any
	switch a := any(arg).(type) {
	case FieldCondition:
		value = a.doc
	case bson.Regex:
		value = a
	}
	return FieldCondition{doc: bson.D{{Key: "$not", Value: value}}}
}

// Or creates a Filter for logical OR: { $or: [ filter1, filter2, ... ] }.
func Or(filters ...Filter) Filter {
	clauses := make(bson.A, 0, len(filters))
	for _, f := range filters {
		clauses = append(clauses, bson.D(f))
	}
	return Filter{{Key: "$or", Value: clauses}}
}

// Polygon creates a legacy polygon geometry ($polygon) from a series of
// coordinate pairs defining the polygon's vertices, for use with GeoWithin.
func Polygon(points ...[]float64) Geometry {
	return geometry{doc: bson.D{{Key: "$polygon", Value: [][]float64(points)}}}
}

// Size creates a FieldCondition matching arrays with the given number of
// elements: { $size: value }.
func Size(value int) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$size", Value: value}}}
}

// Type creates a FieldCondition matching documents where the field is one of the
// specified BSON types: { $type: [ ... ] }. Each type may be an alias string or
// numeric code. The verbose array form is always emitted.
func Type(types ...any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$type", Value: bson.A(types)}}}
}
