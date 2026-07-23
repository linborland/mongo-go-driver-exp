package mql

import (
	"github.com/mongodb-labs/mongo-go-driver-exp/mql/agg"
	"github.com/mongodb-labs/mongo-go-driver-exp/mql/query"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ExamplePipeline() agg.Pipeline {
	filter := query.And()
	pipeline := agg.Pipeline{
		// A basic filter that cuts out any irrelevant partitions from other parttion sets, etc.
		// This filter also cuts out any partitions that don't have stats
		// on the propertyNames we care about.
		agg.MatchStage(filter),
		// This projection filters out any elements in the properties and
		// filters fields that are not for the propertyNames we care
		// about.
		agg.ProjectStage(
			agg.Compute("properties", agg.Filter("$properties", agg.In("$$this.name", []string{"name1", "name2"}))),
			agg.Compute("filters", agg.Filter("$filters", agg.In("$$this.name", []string{"name1", "name2"}))),
		),
		// This stage essentially performs a nested loop over the
		// properties and filters fields, and merges elements from both
		// arrays whenever they refer to the same property. The result
		// should be a single array that contains an element with stats
		// from properties AND filters, for each propertyName we care
		// about.
		agg.ProjectStage(
			agg.Compute("stats", agg.Map(
				"$properties",
				agg.MergeObjects(
					"$$thisProperty",
					agg.Reduce(
						"$filters",
						bson.D{},
						agg.Cond(
							agg.Eq("$$this.name", "$$thisProperty.name"),
							bson.D{
								{Key: "min", Value: agg.Min("$$value.min", "$$this.min")},
								{Key: "max", Value: agg.Max("$$value.max", "$$this.max")},
								{Key: "truncatedStrings", Value: agg.Or("$$value.truncatedStrings", "$$this.truncatedStrings")},
							},
							"$$value",
						),
					),
				),
				agg.WithMapAs("thisProperty"))),
		),
		// This stage simply unwinds the array we just created in the
		// prior state, so that we can operate on every single statistics
		// documents created for all propertyNames we care about.
		agg.UnwindStage("$stats"),
		// This stage groups over the aforementioned stats documents,
		// grouping their statistics together to create a global set of
		// aggregate stats per propertyName.
		agg.GroupStage(
			"$stats.name",
			agg.Accumulate("sum", agg.SumAccumulator("$stats.valueCount")),
			agg.Accumulate("aggregateCount", agg.SumAccumulator("$stats.aggregateCount")),
			agg.Accumulate("min", agg.MinAccumulator("$stats.min")),
			agg.Accumulate("max", agg.MaxAccumulator("$stats.max")),
			agg.Accumulate("truncatedStrings", agg.MaxAccumulator("$stats.truncatedStrings")),
		),
		// This final stage is a quick calculation of $avg, which we could
		// not compute in just the single $group stage above. This is
		// similar to how ADL distributes and computes $group itself.
		agg.AddFieldsStage(
			agg.Assign("avg", agg.Cond(
				agg.Ne("$aggregateCount", 0),
				agg.Divide("$sum", "$aggregateCount"),
				agg.Null,
			)),
			agg.Assign("name", "$_id"),
		),
	}
	return pipeline
}
