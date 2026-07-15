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

// And creates a Filter for logical AND: { $and: [ filter1, filter2, ... ] }.
func And(filters ...Filter) Filter {
	clauses := make(bson.A, 0, len(filters))
	for _, f := range filters {
		clauses = append(clauses, bson.D(f))
	}
	return Filter{{Key: "$and", Value: clauses}}
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

// Ne creates a FieldCondition matching values not equal to value: { $ne: value }.
func Ne(value any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$ne", Value: value}}}
}

// Nin creates a FieldCondition matching none of the given values: { $nin: [ ... ] }.
func Nin(values ...any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$nin", Value: bson.A(values)}}}
}

// Type creates a FieldCondition matching documents where the field is one of the
// specified BSON types: { $type: [ ... ] }. Each type may be an alias string or
// numeric code. The verbose array form is always emitted.
func Type(types ...any) FieldCondition {
	return FieldCondition{doc: bson.D{{Key: "$type", Value: bson.A(types)}}}
}

// Or creates a Filter for logical OR: { $or: [ filter1, filter2, ... ] }.
func Or(filters ...Filter) Filter {
	clauses := make(bson.A, 0, len(filters))
	for _, f := range filters {
		clauses = append(clauses, bson.D(f))
	}
	return Filter{{Key: "$or", Value: clauses}}
}
