package agg_test

import (
	"testing"

	"github.com/mongodb-labs/mongo-go-driver-exp/mql/agg"
	"github.com/mongodb-labs/mongo-go-driver-exp/mql/query"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- $accumulator ---

func TestCustomAccumulator_ImplementAvgOperator(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$author",
			agg.Accumulate("avgCopies",
				agg.CustomAccumulator(
					`function() {
    return { count: 0, sum: 0 }
}`,
					`function(state, numCopies) {
    return { count: state.count + 1, sum: state.sum + numCopies }
}`,
					agg.Array([]string{"$copies"}),
					`function(state1, state2) {
    return {
        count: state1.count + state2.count,
        sum: state1.sum + state2.sum
    }
}`,
					agg.WithCustomFinalize(`function(state) {
    return (state.sum / state.count)
}`),
				),
			),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$author"},
			{Key: "avgCopies", Value: bson.D{{Key: "$accumulator", Value: bson.D{
				{Key: "init", Value: "function() {\n    return { count: 0, sum: 0 }\n}"},
				{Key: "accumulate", Value: "function(state, numCopies) {\n    return { count: state.count + 1, sum: state.sum + numCopies }\n}"},
				{Key: "accumulateArgs", Value: bson.A{"$copies"}},
				{Key: "merge", Value: "function(state1, state2) {\n    return {\n        count: state1.count + state2.count,\n        sum: state1.sum + state2.sum\n    }\n}"},
				{Key: "finalize", Value: "function(state) {\n    return (state.sum / state.count)\n}"},
				{Key: "lang", Value: "js"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestCustomAccumulator_VaryInitialStateByGroup(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{{Key: "city", Value: "$city"}},
			agg.Accumulate("restaurants",
				agg.CustomAccumulator(
					`function(city, userProfileCity) {
    return { max: city === userProfileCity ? 3 : 1, restaurants: [] }
}`,
					`function(state, restaurantName) {
    if (state.restaurants.length < state.max) {
        state.restaurants.push(restaurantName);
    }
    return state;
}`,
					agg.Array([]string{"$name"}),
					`function(state1, state2) {
    return {
        max: state1.max,
        restaurants: state1.restaurants.concat(state2.restaurants).slice(0, state1.max)
    }
}`,
					agg.WithCustomInitArgs("$city", "Bettles"),
					agg.WithCustomFinalize(`function(state) {
    return state.restaurants
}`),
				),
			),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "city", Value: "$city"}}},
			{Key: "restaurants", Value: bson.D{{Key: "$accumulator", Value: bson.D{
				{Key: "init", Value: "function(city, userProfileCity) {\n    return { max: city === userProfileCity ? 3 : 1, restaurants: [] }\n}"},
				{Key: "initArgs", Value: bson.A{"$city", "Bettles"}},
				{Key: "accumulate", Value: "function(state, restaurantName) {\n    if (state.restaurants.length < state.max) {\n        state.restaurants.push(restaurantName);\n    }\n    return state;\n}"},
				{Key: "accumulateArgs", Value: bson.A{"$name"}},
				{Key: "merge", Value: "function(state1, state2) {\n    return {\n        max: state1.max,\n        restaurants: state1.restaurants.concat(state2.restaurants).slice(0, state1.max)\n    }\n}"},
				{Key: "finalize", Value: "function(state) {\n    return state.restaurants\n}"},
				{Key: "lang", Value: "js"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestCustomAccumulator_WithCustomLang(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$author",
			agg.Accumulate("count",
				agg.CustomAccumulator(
					"function() { return 0 }",
					"function(state) { return state + 1 }",
					agg.Array([]string{}),
					"function(state1, state2) { return state1 + state2 }",
					agg.WithCustomLang("js"),
				),
			),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$author"},
			{Key: "count", Value: bson.D{{Key: "$accumulator", Value: bson.D{
				{Key: "init", Value: "function() { return 0 }"},
				{Key: "accumulate", Value: "function(state) { return state + 1 }"},
				{Key: "accumulateArgs", Value: bson.A{}},
				{Key: "merge", Value: "function(state1, state2) { return state1 + state2 }"},
				{Key: "lang", Value: "js"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $addToSet ---

func TestAddToSetAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{
				{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
			},
			agg.Accumulate("itemsSold", agg.AddToSetAccumulator("$item")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
			}},
			{Key: "itemsSold", Value: bson.D{{Key: "$addToSet", Value: "$item"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestAddToSetAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("cakeTypesForState", agg.AddToSetAccumulator("$type"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "cakeTypesForState", Value: bson.D{
					{Key: "$addToSet", Value: "$type"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $avg ---

func TestAvgAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$item",
			agg.Accumulate("avgAmount", agg.AvgAccumulator(agg.Multiply("$price", "$quantity"))),
			agg.Accumulate("avgQuantity", agg.AvgAccumulator("$quantity")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$item"},
			{Key: "avgAmount", Value: bson.D{{Key: "$avg", Value: bson.D{{Key: "$multiply", Value: bson.A{"$price", "$quantity"}}}}}},
			{Key: "avgQuantity", Value: bson.D{{Key: "$avg", Value: "$quantity"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestAvgAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("averageQuantityForState", agg.AvgAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "averageQuantityForState", Value: bson.D{
					{Key: "$avg", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $bottom ---

func TestBottomAccumulator_FindBottomScore(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.BottomAccumulator(
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$bottom", Value: bson.D{
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: bson.A{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestBottomAccumulator_FindBottomScoreAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.BottomAccumulator(
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$bottom", Value: bson.D{
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: bson.A{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $bottomN ---

func TestBottomNAccumulator_FindThreeLowestScores(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.BottomNAccumulator(
				3,
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$bottomN", Value: bson.D{
				{Key: "n", Value: 3},
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: []string{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestBottomNAccumulator_FindThreeLowestScoreDocsAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.BottomNAccumulator(
				3,
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$bottomN", Value: bson.D{
				{Key: "n", Value: 3},
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: []string{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $concatArrays ---

func TestConcatArraysAccumulator_WarehouseCollection(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$location",
			agg.Accumulate("array", agg.ConcatArraysAccumulator("$items")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$location"},
			{Key: "array", Value: bson.D{{Key: "$concatArrays", Value: "$items"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $count ---
func TestCountAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$state",
			agg.Accumulate("countNumberOfDocumentsForState", agg.CountAccumulator()),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$state"},
			{Key: "countNumberOfDocumentsForState", Value: bson.D{{Key: "$count", Value: bson.D{}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestCountAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("countNumberOfDocumentsForState", agg.CountAccumulator(),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "countNumberOfDocumentsForState", Value: bson.D{
					{Key: "$count", Value: bson.D{}},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $covariancePop ---

func TestCovariancePopAccumulator_Example(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("covariancePopForState",
					agg.CovariancePopAccumulator(agg.Year("$orderDate"), "$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "covariancePopForState", Value: bson.D{
					{Key: "$covariancePop", Value: bson.A{
						bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$orderDate"}}}},
						"$quantity",
					}},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $covarianceSamp ---

func TestCovarianceSampAccumulator_Example(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("covarianceSampForState",
					agg.CovarianceSampAccumulator(agg.Year("$orderDate"), "$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "covarianceSampForState", Value: bson.D{
					{Key: "$covarianceSamp", Value: bson.A{
						bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$orderDate"}}}},
						"$quantity",
					}},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $denseRank ---

func TestDenseRankAccumulator_DenseRankPartitionsByAnIntegerField(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("denseRankQuantityForState", agg.DenseRankAccumulator()),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("quantity", agg.Desc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "quantity", Value: int32(-1)}}},
			{Key: "output", Value: bson.D{
				{Key: "denseRankQuantityForState", Value: bson.D{{Key: "$denseRank", Value: bson.D{}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestDenseRankAccumulator_DenseRankPartitionsByADateField(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("denseRankOrderDateForState", agg.DenseRankAccumulator()),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "denseRankOrderDateForState", Value: bson.D{{Key: "$denseRank", Value: bson.D{}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $derivative ---

func TestDerivativeAccumulator_Example(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("truckAverageSpeed",
					agg.DerivativeAccumulator("$miles", agg.WithDerivativeUnit("hour")),
					agg.WithWindowRange(agg.WindowOffset(-30), agg.WindowOffset(0)),
					agg.WithWindowRangeUnit("second")),
			},
			agg.WithSetWindowFieldsPartitionBy("$truckID"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("timeStamp", agg.Asc)),
		),
		agg.MatchStage(query.Field("truckAverageSpeed", query.Gt(50))),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$truckID"},
			{Key: "sortBy", Value: bson.D{{Key: "timeStamp", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "truckAverageSpeed", Value: bson.D{
					{Key: "$derivative", Value: bson.D{
						{Key: "input", Value: "$miles"},
						{Key: "unit", Value: "hour"},
					}},
					{Key: "window", Value: bson.D{
						{Key: "range", Value: bson.A{-30, 0}},
						{Key: "unit", Value: "second"},
					}},
				}},
			}},
		}}},
		bson.D{{Key: "$match", Value: bson.D{{Key: "truckAverageSpeed", Value: bson.D{{Key: "$gt", Value: 50}}}}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $documentNumber ---

func TestDocumentNumberAccumulator_DocumentNumberForEachState(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("documentNumberForState", agg.DocumentNumberAccumulator()),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("quantity", agg.Desc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "quantity", Value: int32(-1)}}},
			{Key: "output", Value: bson.D{
				{Key: "documentNumberForState", Value: bson.D{{Key: "$documentNumber", Value: bson.D{}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $expMovingAvg ---

func TestExpMovingAvgAccumulator_ExponentialMovingAverageUsingN(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("expMovingAvgForStock", agg.ExpMovingAvgNAccumulator("$price", 2)),
			},
			agg.WithSetWindowFieldsPartitionBy("$stock"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("date", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$stock"},
			{Key: "sortBy", Value: bson.D{{Key: "date", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "expMovingAvgForStock", Value: bson.D{{Key: "$expMovingAvg", Value: bson.D{
					{Key: "input", Value: "$price"},
					{Key: "N", Value: int32(2)},
				}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestExpMovingAvgAccumulator_ExponentialMovingAverageUsingAlpha(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("expMovingAvgForStock", agg.ExpMovingAvgAlphaAccumulator("$price", 0.75)),
			},
			agg.WithSetWindowFieldsPartitionBy("$stock"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("date", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$stock"},
			{Key: "sortBy", Value: bson.D{{Key: "date", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "expMovingAvgForStock", Value: bson.D{{Key: "$expMovingAvg", Value: bson.D{
					{Key: "input", Value: "$price"},
					{Key: "alpha", Value: 0.75},
				}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $first ---

func TestFirstAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SortStage(agg.Sort("item", agg.Asc), agg.Sort("date", agg.Asc)),
		agg.GroupStage(
			"$item",
			agg.Accumulate("firstSale", agg.FirstAccumulator("$date")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "item", Value: 1},
			{Key: "date", Value: 1},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$item"},
			{Key: "firstSale", Value: bson.D{{Key: "$first", Value: "$date"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestFirstAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("firstOrderTypeForState", agg.FirstAccumulator("$type"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "firstOrderTypeForState", Value: bson.D{
					{Key: "$first", Value: "$type"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $firstN ---

func TestFirstNAccumulator_NullAndMissingValues(t *testing.T) {
	got := agg.Pipeline{
		agg.DocumentsStage(bson.A{
			bson.D{{Key: "playerId", Value: "PlayerA"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: 1}},
			bson.D{{Key: "playerId", Value: "PlayerB"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: 2}},
			bson.D{{Key: "playerId", Value: "PlayerC"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: 3}},
			bson.D{{Key: "playerId", Value: "PlayerD"}, {Key: "gameId", Value: "G1"}},
			bson.D{{Key: "playerId", Value: "PlayerE"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: nil}},
		}),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("firstFiveScores", agg.FirstNAccumulator("$score", 5)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$documents", Value: bson.A{
			bson.D{{Key: "playerId", Value: "PlayerA"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: 1}},
			bson.D{{Key: "playerId", Value: "PlayerB"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: 2}},
			bson.D{{Key: "playerId", Value: "PlayerC"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: 3}},
			bson.D{{Key: "playerId", Value: "PlayerD"}, {Key: "gameId", Value: "G1"}},
			bson.D{{Key: "playerId", Value: "PlayerE"}, {Key: "gameId", Value: "G1"}, {Key: "score", Value: nil}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "firstFiveScores", Value: bson.D{{Key: "$firstN", Value: bson.D{
				{Key: "input", Value: "$score"},
				{Key: "n", Value: 5},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestFirstNAccumulator_FindFirstThreePlayerScoresForSingleGame(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("firstThreeScores", agg.FirstNAccumulator(
				[]string{"$playerId", "$score"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "firstThreeScores", Value: bson.D{{Key: "$firstN", Value: bson.D{
				{Key: "input", Value: []string{"$playerId", "$score"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestFirstNAccumulator_FindFirstThreePlayerScoresAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.FirstNAccumulator(
				[]string{"$playerId", "$score"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$firstN", Value: bson.D{
				{Key: "input", Value: []string{"$playerId", "$score"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestFirstNAccumulator_UsingSortWithFirstN(t *testing.T) {
	got := agg.Pipeline{
		agg.SortStage(agg.Sort("score", agg.Desc)),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.FirstNAccumulator(
				[]string{"$playerId", "$score"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$sort", Value: bson.D{{Key: "score", Value: -1}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$firstN", Value: bson.D{
				{Key: "input", Value: []string{"$playerId", "$score"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestFirstNAccumulator_ComputeNBasedOnGroupKey(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{{Key: "gameId", Value: "$gameId"}},
			agg.Accumulate("gamescores", agg.FirstNAccumulator(
				"$score",
				agg.Cond(agg.Eq("$gameId", "G2"), 1, 3),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "gameId", Value: "$gameId"}}},
			{Key: "gamescores", Value: bson.D{{Key: "$firstN", Value: bson.D{
				{Key: "input", Value: "$score"},
				{Key: "n", Value: bson.D{{Key: "$cond", Value: bson.D{
					{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{"$gameId", "G2"}}}},
					{Key: "then", Value: 1},
					{Key: "else", Value: 3},
				}}}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $integral ---

func TestIntegralAccumulator_Example(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("powerMeterKilowattHours",
					agg.IntegralAccumulator("$kilowatts", agg.WithIntegralUnit("hour")),
					agg.WithWindowRange(agg.WindowUnbounded, agg.WindowCurrent),
					agg.WithWindowRangeUnit("hour")),
			},
			agg.WithSetWindowFieldsPartitionBy("$powerMeterID"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("timeStamp", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$powerMeterID"},
			{Key: "sortBy", Value: bson.D{{Key: "timeStamp", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "powerMeterKilowattHours", Value: bson.D{
					{Key: "$integral", Value: bson.D{
						{Key: "input", Value: "$kilowatts"},
						{Key: "unit", Value: "hour"},
					}},
					{Key: "window", Value: bson.D{
						{Key: "range", Value: bson.A{"unbounded", "current"}},
						{Key: "unit", Value: "hour"},
					}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $last ---

func TestLastAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SortStage(
			agg.Sort("item", agg.Asc),
			agg.Sort("date", agg.Asc),
		),
		agg.GroupStage(
			"$item",
			agg.Accumulate("lastSalesDate", agg.LastAccumulator("$date")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "item", Value: 1},
			{Key: "date", Value: 1},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$item"},
			{Key: "lastSalesDate", Value: bson.D{{Key: "$last", Value: "$date"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestLastAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("lastOrderTypeForState", agg.LastAccumulator("$type"),
					agg.WithWindowDocuments(agg.WindowCurrent, agg.WindowUnbounded)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "lastOrderTypeForState", Value: bson.D{
					{Key: "$last", Value: "$type"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"current", "unbounded"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $lastN ---

func TestLastNAccumulator_FindLastThreePlayerScoresForSingleGame(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("lastThreeScores", agg.LastNAccumulator(
				[]string{"$playerId", "$score"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "lastThreeScores", Value: bson.D{{Key: "$lastN", Value: bson.D{
				{Key: "input", Value: []string{"$playerId", "$score"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestLastNAccumulator_FindLastThreePlayerScoresAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.LastNAccumulator(
				[]string{"$playerId", "$score"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$lastN", Value: bson.D{
				{Key: "input", Value: []string{"$playerId", "$score"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestLastNAccumulator_UsingSortWithLastN(t *testing.T) {
	got := agg.Pipeline{
		agg.SortStage(agg.Sort("score", agg.Desc)),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.LastNAccumulator(
				[]string{"$playerId", "$score"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$sort", Value: bson.D{{Key: "score", Value: -1}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$lastN", Value: bson.D{
				{Key: "input", Value: []string{"$playerId", "$score"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestLastNAccumulator_ComputeNBasedOnGroupKey(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{{Key: "gameId", Value: "$gameId"}},
			agg.Accumulate("gamescores", agg.LastNAccumulator(
				"$score",
				agg.Cond(agg.Eq("$gameId", "G2"), 1, 3),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "gameId", Value: "$gameId"}}},
			{Key: "gamescores", Value: bson.D{{Key: "$lastN", Value: bson.D{
				{Key: "input", Value: "$score"},
				{Key: "n", Value: bson.D{{Key: "$cond", Value: bson.D{
					{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{"$gameId", "G2"}}}},
					{Key: "then", Value: 1},
					{Key: "else", Value: 3},
				}}}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $linearFill ---

func TestLinearFillAccumulator_FillMissingValuesWithLinearInterpolation(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("price", agg.LinearFillAccumulator("$price")),
			},
			agg.WithSetWindowFieldsSortBy(agg.Sort("time", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "time", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "price", Value: bson.D{{Key: "$linearFill", Value: "$price"}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestLinearFillAccumulator_UseMultipleFillMethodsInASingleStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("linearFillPrice", agg.LinearFillAccumulator("$price")),
				agg.WindowOutput("locfPrice", agg.LocfAccumulator("$price")),
			},
			agg.WithSetWindowFieldsSortBy(agg.Sort("time", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "time", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "linearFillPrice", Value: bson.D{{Key: "$linearFill", Value: "$price"}}},
				{Key: "locfPrice", Value: bson.D{{Key: "$locf", Value: "$price"}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $locf ---

func TestLocfAccumulator_FillMissingValuesWithTheLastObservedValue(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("price", agg.LocfAccumulator("$price")),
			},
			agg.WithSetWindowFieldsSortBy(agg.Sort("time", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "time", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "price", Value: bson.D{{Key: "$locf", Value: "$price"}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $max ---

func TestMaxAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$item",
			agg.Accumulate("maxTotalAmount", agg.MaxAccumulator(agg.Multiply("$price", "$quantity"))),
			agg.Accumulate("maxQuantity", agg.MaxAccumulator("$quantity")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$item"},
			{Key: "maxTotalAmount", Value: bson.D{{Key: "$max", Value: bson.D{{Key: "$multiply", Value: bson.A{"$price", "$quantity"}}}}}},
			{Key: "maxQuantity", Value: bson.D{{Key: "$max", Value: "$quantity"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMaxAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("maximumQuantityForState", agg.MaxAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "maximumQuantityForState", Value: bson.D{
					{Key: "$max", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $maxN ---

func TestMaxNAccumulator_FindMaxThreeScoresForSingleGame(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("maxThreeScores", agg.MaxNAccumulator(
				[]string{"$score", "$playerId"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "maxThreeScores", Value: bson.D{{Key: "$maxN", Value: bson.D{
				{Key: "input", Value: []string{"$score", "$playerId"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMaxNAccumulator_FindMaxThreeScoresAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("maxScores", agg.MaxNAccumulator(
				[]string{"$score", "$playerId"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "maxScores", Value: bson.D{{Key: "$maxN", Value: bson.D{
				{Key: "input", Value: []string{"$score", "$playerId"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMaxNAccumulator_ComputeNBasedOnGroupKey(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{{Key: "gameId", Value: "$gameId"}},
			agg.Accumulate("gamescores", agg.MaxNAccumulator(
				[]string{"$score", "$playerId"},
				agg.Cond(agg.Eq("$gameId", "G2"), 1, 3),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "gameId", Value: "$gameId"}}},
			{Key: "gamescores", Value: bson.D{{Key: "$maxN", Value: bson.D{
				{Key: "input", Value: []string{"$score", "$playerId"}},
				{Key: "n", Value: bson.D{{Key: "$cond", Value: bson.D{
					{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{"$gameId", "G2"}}}},
					{Key: "then", Value: 1},
					{Key: "else", Value: 3},
				}}}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $median ---

func TestMedianAccumulator_UseMedianAsAnAccumulator(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			agg.Null,
			agg.Accumulate("test01_median", agg.MedianAccumulator(
				"$test01",
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "test01_median", Value: bson.D{{Key: "$median", Value: bson.D{
				{Key: "input", Value: "$test01"},
				{Key: "method", Value: "approximate"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMedianAccumulator_UseMedianInSetWindowFieldStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("test01_median", agg.MedianAccumulator("$test01"),
					agg.WithWindowRange(agg.WindowOffset(-3), agg.WindowOffset(3))),
			},
			agg.WithSetWindowFieldsSortBy(agg.Sort("test01", agg.Asc)),
		),
		agg.ProjectStage(
			agg.Exclude("_id"),
			agg.Include("studentId"),
			agg.Include("test01_median"),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "test01", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "test01_median", Value: bson.D{
					{Key: "$median", Value: bson.D{
						{Key: "input", Value: "$test01"},
						{Key: "method", Value: "approximate"},
					}},
					{Key: "window", Value: bson.D{{Key: "range", Value: bson.A{-3, 3}}}},
				}},
			}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: int32(0)},
			{Key: "studentId", Value: int32(1)},
			{Key: "test01_median", Value: int32(1)},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $mergeObjects ---

func TestMergeObjectsAccumulator_MergeObjectsAsAnAccumulator(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$item",
			agg.Accumulate("mergedSales", agg.MergeObjectsAccumulator(
				"$quantity",
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$item"},
			{Key: "mergedSales", Value: bson.D{{Key: "$mergeObjects", Value: "$quantity"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $min ---

func TestMinAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$item",
			agg.Accumulate("minQuantity", agg.MinAccumulator("$quantity")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$item"},
			{Key: "minQuantity", Value: bson.D{{Key: "$min", Value: "$quantity"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMinAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("minimumQuantityForState", agg.MinAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "minimumQuantityForState", Value: bson.D{
					{Key: "$min", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $minMaxScaler ---

func TestMinMaxScalerAccumulator_NormalizeValuesWithCustomRange(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("scaled", agg.MinMaxScalerAccumulator("$a")),
				agg.WindowOutput("scaledTo100", agg.MinMaxScalerRangeAccumulator("$a", 0, 100)),
			},
			agg.WithSetWindowFieldsSortBy(agg.Sort("a", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "a", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "scaled", Value: bson.D{{Key: "$minMaxScaler", Value: bson.D{
					{Key: "input", Value: "$a"},
				}}}},
				{Key: "scaledTo100", Value: bson.D{{Key: "$minMaxScaler", Value: bson.D{
					{Key: "input", Value: "$a"},
					{Key: "min", Value: 0},
					{Key: "max", Value: 100},
				}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $minN ---

func TestMinNAccumulator_FindMinThreeScoresForSingleGame(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("minScores", agg.MinNAccumulator(
				[]string{"$score", "$playerId"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "minScores", Value: bson.D{{Key: "$minN", Value: bson.D{
				{Key: "input", Value: []string{"$score", "$playerId"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMinNAccumulator_FindMinThreeDocumentsAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("minScores", agg.MinNAccumulator(
				[]string{"$score", "$playerId"},
				3,
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "minScores", Value: bson.D{{Key: "$minN", Value: bson.D{
				{Key: "input", Value: []string{"$score", "$playerId"}},
				{Key: "n", Value: 3},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestMinNAccumulator_ComputeNBasedOnGroupKey(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{{Key: "gameId", Value: "$gameId"}},
			agg.Accumulate("gamescores", agg.MinNAccumulator(
				[]string{"$score", "$playerId"},
				agg.Cond(agg.Eq("$gameId", "G2"), 1, 3),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "gameId", Value: "$gameId"}}},
			{Key: "gamescores", Value: bson.D{{Key: "$minN", Value: bson.D{
				{Key: "input", Value: []string{"$score", "$playerId"}},
				{Key: "n", Value: bson.D{{Key: "$cond", Value: bson.D{
					{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{"$gameId", "G2"}}}},
					{Key: "then", Value: 1},
					{Key: "else", Value: 3},
				}}}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $percentile ---

func TestPercentileAccumulator_CalculateSingleValueAsAccumulator(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			agg.Null,
			agg.Accumulate("test01_percentiles", agg.PercentileAccumulator(
				"$test01",
				[]float64{0.95},
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "test01_percentiles", Value: bson.D{{Key: "$percentile", Value: bson.D{
				{Key: "input", Value: "$test01"},
				{Key: "p", Value: []float64{0.95}},
				{Key: "method", Value: "approximate"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestPercentileAccumulator_CalculateMultipleValuesAsAccumulator(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			agg.Null,
			agg.Accumulate("test01_percentiles", agg.PercentileAccumulator(
				"$test01",
				[]float64{0.5, 0.75, 0.9, 0.95},
			)),
			agg.Accumulate("test02_percentiles", agg.PercentileAccumulator(
				"$test02",
				[]float64{0.5, 0.75, 0.9, 0.95},
			)),
			agg.Accumulate("test03_percentiles", agg.PercentileAccumulator(
				"$test03",
				[]float64{0.5, 0.75, 0.9, 0.95},
			)),
			agg.Accumulate("test03_percent_alt", agg.PercentileAccumulator(
				"$test03",
				[]float64{0.9, 0.5, 0.75, 0.95},
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "test01_percentiles", Value: bson.D{{Key: "$percentile", Value: bson.D{
				{Key: "input", Value: "$test01"},
				{Key: "p", Value: []float64{0.5, 0.75, 0.9, 0.95}},
				{Key: "method", Value: "approximate"},
			}}}},
			{Key: "test02_percentiles", Value: bson.D{{Key: "$percentile", Value: bson.D{
				{Key: "input", Value: "$test02"},
				{Key: "p", Value: []float64{0.5, 0.75, 0.9, 0.95}},
				{Key: "method", Value: "approximate"},
			}}}},
			{Key: "test03_percentiles", Value: bson.D{{Key: "$percentile", Value: bson.D{
				{Key: "input", Value: "$test03"},
				{Key: "p", Value: []float64{0.5, 0.75, 0.9, 0.95}},
				{Key: "method", Value: "approximate"},
			}}}},
			{Key: "test03_percent_alt", Value: bson.D{{Key: "$percentile", Value: bson.D{
				{Key: "input", Value: "$test03"},
				{Key: "p", Value: []float64{0.9, 0.5, 0.75, 0.95}},
				{Key: "method", Value: "approximate"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestPercentileAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("test01_95percentile",
					agg.PercentileAccumulator("$test01", []float64{0.95}),
					agg.WithWindowRange(agg.WindowOffset(-3), agg.WindowOffset(3))),
			},
			agg.WithSetWindowFieldsSortBy(agg.Sort("test01", agg.Asc)),
		),
		agg.ProjectStage(
			agg.Exclude("_id"),
			agg.Include("studentId"),
			agg.Include("test01_95percentile"),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "sortBy", Value: bson.D{{Key: "test01", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "test01_95percentile", Value: bson.D{
					{Key: "$percentile", Value: bson.D{
						{Key: "input", Value: "$test01"},
						{Key: "p", Value: []float64{0.95}},
						{Key: "method", Value: "approximate"},
					}},
					{Key: "window", Value: bson.D{{Key: "range", Value: bson.A{-3, 3}}}},
				}},
			}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: int32(0)},
			{Key: "studentId", Value: int32(1)},
			{Key: "test01_95percentile", Value: int32(1)},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $push ---

func TestPushAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SortStage(
			agg.Sort("date", agg.Asc),
			agg.Sort("item", agg.Asc),
		),
		agg.GroupStage(
			bson.D{
				{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
			},
			agg.Accumulate("itemsSold", agg.PushAccumulator(bson.D{
				{Key: "item", Value: "$item"},
				{Key: "quantity", Value: "$quantity"},
			})),
		),
	}
	want := bson.A{
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "date", Value: 1},
			{Key: "item", Value: 1},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
			}},
			{Key: "itemsSold", Value: bson.D{{Key: "$push", Value: bson.D{
				{Key: "item", Value: "$item"},
				{Key: "quantity", Value: "$quantity"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestPushAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("quantitiesForState", agg.PushAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "quantitiesForState", Value: bson.D{
					{Key: "$push", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $rank ---

func TestRankAccumulator_RankPartitionsByAnIntegerField(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("rankQuantityForState", agg.RankAccumulator()),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("quantity", agg.Desc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "quantity", Value: int32(-1)}}},
			{Key: "output", Value: bson.D{
				{Key: "rankQuantityForState", Value: bson.D{{Key: "$rank", Value: bson.D{}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestRankAccumulator_RankPartitionsByADateField(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("rankOrderDateForState", agg.RankAccumulator()),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "rankOrderDateForState", Value: bson.D{{Key: "$rank", Value: bson.D{}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $setUnion ---

func TestSetUnionAccumulator_FlowersCollection(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$location",
			agg.Accumulate("allFlowers", agg.SetUnionAccumulator("$flowers")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$location"},
			{Key: "allFlowers", Value: bson.D{{Key: "$setUnion", Value: "$flowers"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestSetUnionAccumulator_FlowersCollectionProjection(t *testing.T) {
	got := agg.Pipeline{
		agg.ProjectStage(
			agg.Include("flowerFieldA"),
			agg.Include("flowerFieldB"),
			agg.Compute("allValues", agg.SetUnion("$flowerFieldA", "$flowerFieldB")),
			agg.Exclude("_id"),
		),
	}
	want := bson.A{
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "flowerFieldA", Value: int32(1)},
			{Key: "flowerFieldB", Value: int32(1)},
			{Key: "allValues", Value: bson.D{{Key: "$setUnion", Value: bson.A{"$flowerFieldA", "$flowerFieldB"}}}},
			{Key: "_id", Value: int32(0)},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $shift ---

func TestShiftAccumulator_ShiftUsingPositiveInteger(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("shiftQuantityForState",
					agg.ShiftAccumulator("$quantity", 1, agg.WithShiftDefault("Not available"))),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("quantity", agg.Desc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "quantity", Value: int32(-1)}}},
			{Key: "output", Value: bson.D{
				{Key: "shiftQuantityForState", Value: bson.D{{Key: "$shift", Value: bson.D{
					{Key: "output", Value: "$quantity"},
					{Key: "by", Value: int32(1)},
					{Key: "default", Value: "Not available"},
				}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestShiftAccumulator_ShiftUsingNegativeInteger(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("shiftQuantityForState",
					agg.ShiftAccumulator("$quantity", -1, agg.WithShiftDefault("Not available"))),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("quantity", agg.Desc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "quantity", Value: int32(-1)}}},
			{Key: "output", Value: bson.D{
				{Key: "shiftQuantityForState", Value: bson.D{{Key: "$shift", Value: bson.D{
					{Key: "output", Value: "$quantity"},
					{Key: "by", Value: int32(-1)},
					{Key: "default", Value: "Not available"},
				}}}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $stdDevPop ---

func TestStdDevPopAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$quiz",
			agg.Accumulate("stdDev", agg.StdDevPopAccumulator("$score")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$quiz"},
			{Key: "stdDev", Value: bson.D{{Key: "$stdDevPop", Value: "$score"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestStdDevPopAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("stdDevPopQuantityForState", agg.StdDevPopAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "stdDevPopQuantityForState", Value: bson.D{
					{Key: "$stdDevPop", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $stdDevSamp ---

// TODO: omits the leading $sample stage (size: 100) as $sample is not yet implemented
func TestStdDevSampAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			agg.Null,
			agg.Accumulate("ageStdDev", agg.StdDevSampAccumulator("$age")),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "ageStdDev", Value: bson.D{{Key: "$stdDevSamp", Value: "$age"}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestStdDevSampAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("stdDevSampQuantityForState", agg.StdDevSampAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "stdDevSampQuantityForState", Value: bson.D{
					{Key: "$stdDevSamp", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $sum ---

func TestSumAccumulator_UseInGroupStage(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{
				{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
			},
			agg.Accumulate("totalAmount", agg.SumAccumulator(agg.Multiply("$price", "$quantity"))),
			agg.Accumulate("count", agg.SumAccumulator(1)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
			}},
			{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{"$price", "$quantity"}}}}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestSumAccumulator_UseInSetWindowFieldsStage(t *testing.T) {
	got := agg.Pipeline{
		agg.SetWindowFieldsStage(
			[]agg.WindowField{
				agg.WindowOutput("sumQuantityForState", agg.SumAccumulator("$quantity"),
					agg.WithWindowDocuments(agg.WindowUnbounded, agg.WindowCurrent)),
			},
			agg.WithSetWindowFieldsPartitionBy("$state"),
			agg.WithSetWindowFieldsSortBy(agg.Sort("orderDate", agg.Asc)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$setWindowFields", Value: bson.D{
			{Key: "partitionBy", Value: "$state"},
			{Key: "sortBy", Value: bson.D{{Key: "orderDate", Value: int32(1)}}},
			{Key: "output", Value: bson.D{
				{Key: "sumQuantityForState", Value: bson.D{
					{Key: "$sum", Value: "$quantity"},
					{Key: "window", Value: bson.D{{Key: "documents", Value: bson.A{"unbounded", "current"}}}},
				}},
			}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $top ---

func TestTopAccumulator_FindTopScore(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.TopAccumulator(
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$top", Value: bson.D{
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: bson.A{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestTopAccumulator_FindTopScoreAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.TopAccumulator(
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$top", Value: bson.D{
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: bson.A{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

// --- $topN ---

func TestTopNAccumulator_FindThreeHighestScores(t *testing.T) {
	got := agg.Pipeline{
		agg.MatchStage(query.Field("gameId", query.Eq("G1"))),
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.TopNAccumulator(
				3,
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: bson.D{{Key: "$eq", Value: "G1"}}}}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$topN", Value: bson.D{
				{Key: "n", Value: 3},
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: bson.A{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestTopNAccumulator_FindThreeHighestScoreDocsAcrossMultipleGames(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			"$gameId",
			agg.Accumulate("playerId", agg.TopNAccumulator(
				3,
				[]string{"$playerId", "$score"},
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$gameId"},
			{Key: "playerId", Value: bson.D{{Key: "$topN", Value: bson.D{
				{Key: "n", Value: 3},
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: bson.A{"$playerId", "$score"}},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}

func TestTopNAccumulator_ComputingNBasedOnGroupKey(t *testing.T) {
	got := agg.Pipeline{
		agg.GroupStage(
			bson.D{{Key: "gameId", Value: "$gameId"}},
			agg.Accumulate("gamescores", agg.TopNAccumulator(
				agg.Cond(agg.Eq("$gameId", "G2"), 1, 3),
				"$score",
				agg.Sort("score", agg.Desc),
			)),
		),
	}
	want := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "gameId", Value: "$gameId"}}},
			{Key: "gamescores", Value: bson.D{{Key: "$topN", Value: bson.D{
				{Key: "n", Value: bson.D{{Key: "$cond", Value: bson.D{
					{Key: "if", Value: bson.D{{Key: "$eq", Value: bson.A{"$gameId", "G2"}}}},
					{Key: "then", Value: 1},
					{Key: "else", Value: 3},
				}}}},
				{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
				{Key: "output", Value: "$score"},
			}}}},
		}}},
	}
	assertPipelineEqual(t, got, want)
}
