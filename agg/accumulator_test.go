package agg_test

import (
	"testing"

	"github.com/mongodb-labs/mongo-go-driver-exp/agg"
	"github.com/mongodb-labs/mongo-go-driver-exp/agg/query"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- $accumulator ---

func TestCustomAccumulator_ImplementAvgOperator(t *testing.T) {
	finalize := `function(state) {
    return (state.sum / state.count)
}`
	assertPipelineEqual(t,
		agg.Pipeline{
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
						"js",
						nil,
						&finalize,
					),
				),
			),
		},
		bson.A{
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
		},
	)
}

func TestCustomAccumulator_VaryInitialStateByGroup(t *testing.T) {
	finalize := `function(state) {
    return state.restaurants
}`
	assertPipelineEqual(t,
		agg.Pipeline{
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
						"js",
						[]any{"$city", "Bettles"},
						&finalize,
					),
				),
			),
		},
		bson.A{
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
		},
	)
}

// --- $addToSet ---

func TestAddToSetAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				bson.D{
					{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
					{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				},
				agg.Accumulate("itemsSold", agg.AddToSetAccumulator("$item")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
					{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				}},
				{Key: "itemsSold", Value: bson.D{{Key: "$addToSet", Value: "$item"}}},
			}}},
		},
	)
}

// TODO: implement TestAddToSetAccumulator_UseInSetWindowFieldsStage
// after SetWindowFieldsStage is implemented

// --- $avg ---

func TestAvgAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$item",
				agg.Accumulate("avgAmount", agg.AvgAccumulator(agg.Multiply("$price", "$quantity"))),
				agg.Accumulate("avgQuantity", agg.AvgAccumulator("$quantity")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$item"},
				{Key: "avgAmount", Value: bson.D{{Key: "$avg", Value: bson.D{{Key: "$multiply", Value: bson.A{"$price", "$quantity"}}}}}},
				{Key: "avgQuantity", Value: bson.D{{Key: "$avg", Value: "$quantity"}}},
			}}},
		},
	)
}

// TODO: implement TestAvgAccumulator_UseInSetWindowFieldsStage
// after SetWindowFieldsStage is implemented

// --- $bottom ---

func TestBottomAccumulator_FindBottomScore(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.BottomAccumulator(
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$bottom", Value: bson.D{
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: bson.A{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

func TestBottomAccumulator_FindBottomScoreAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.BottomAccumulator(
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$bottom", Value: bson.D{
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: bson.A{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

// --- $bottomN ---

func TestBottomNAccumulator_FindThreeLowestScores(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.BottomNAccumulator(
					3,
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$bottomN", Value: bson.D{
					{Key: "n", Value: 3},
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: []string{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

func TestBottomNAccumulator_FindThreeLowestScoreDocsAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.BottomNAccumulator(
					3,
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$bottomN", Value: bson.D{
					{Key: "n", Value: 3},
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: []string{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestBottomNAccumulator_FindThreeLowestScoreDocsAcrossMultipleGames
// after $cond operator is implemented

// --- $concatArrays ---

func TestConcatArraysAccumulator_WarehouseCollection(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$location",
				agg.Accumulate("array", agg.ConcatArraysAccumulator("$items")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$location"},
				{Key: "array", Value: bson.D{{Key: "$concatArrays", Value: "$items"}}},
			}}},
		},
	)
}

// --- $count ---
func TestCountAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$state",
				agg.Accumulate("countNumberOfDocumentsForState", agg.CountAccumulator()),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$state"},
				{Key: "countNumberOfDocumentsForState", Value: bson.D{{Key: "$count", Value: bson.D{}}}},
			}}},
		},
	)
}

// TODO: implement TestCount_UseInGroupStage
// after SetWindowFieldsStage is implemented

// --- $covariancePop ---

// TODO: implement covariancePop tests
// after SetWindowFieldsStage is implemented

// --- $covarianceSamp ---

// TODO: implement covarianceSamp tests
// after SetWindowFieldsStage is implemented

// --- $denseRank ---

// TODO: implement denseRank tests
// after SetWindowFieldsStage is implemented

// --- $derivative ---

// TODO: implement derivative tests
// after SetWindowFieldsStage is implemented

// --- $documentNumber ---

// TODO: implement documentNumber tests
// after SetWindowFieldsStage is implemented

// --- $expMovingAvg ---

// TODO: implement expMovingAvg tests
// after SetWindowFieldsStage is implemented

// --- $first ---

func TestFirstAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.SortStage(agg.Sort("item", agg.Asc), agg.Sort("date", agg.Asc)),
			agg.GroupStage(
				"$item",
				agg.Accumulate("firstSale", agg.FirstAccumulator("$date")),
			),
		},
		bson.A{
			bson.D{{Key: "$sort", Value: bson.D{
				{Key: "item", Value: 1},
				{Key: "date", Value: 1},
			}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$item"},
				{Key: "firstSale", Value: bson.D{{Key: "$first", Value: "$date"}}},
			}}},
		},
	)
}

// TODO: implement TestFirstAccumulator_UseInSetMovingFieldStage tests
// after SetWindowFieldsStage is implemented

// --- $firstN ---

// TODO: implement TestFirstNAccumulator_NullAndMissingValues
// after $documents stage is implemented

func TestFirstNAccumulator_FindFirstThreePlayerScoresForSingleGame(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("firstThreeScores", agg.FirstNAccumulator(
					[]string{"$playerId", "$score"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "firstThreeScores", Value: bson.D{{Key: "$firstN", Value: bson.D{
					{Key: "input", Value: []string{"$playerId", "$score"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

func TestFirstNAccumulator_FindFirstThreePlayerScoresAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.FirstNAccumulator(
					[]string{"$playerId", "$score"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$firstN", Value: bson.D{
					{Key: "input", Value: []string{"$playerId", "$score"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

func TestFirstNAccumulator_UsingSortWithFirstN(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.SortStage(agg.Sort("score", agg.Desc)),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.FirstNAccumulator(
					[]string{"$playerId", "$score"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$sort", Value: bson.D{{Key: "score", Value: -1}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$firstN", Value: bson.D{
					{Key: "input", Value: []string{"$playerId", "$score"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestFirstNAccumulator_ComputeNBasedOnGroupKey
// after $cond operator is implemented

// --- $integral ---

// TODO: implement integral tests
// after SetWindowFieldsStage is implemented

// --- $last ---

func TestLastAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.SortStage(
				agg.Sort("item", agg.Asc),
				agg.Sort("date", agg.Asc),
			),
			agg.GroupStage(
				"$item",
				agg.Accumulate("lastSalesDate", agg.LastAccumulator("$date")),
			),
		},
		bson.A{
			bson.D{{Key: "$sort", Value: bson.D{
				{Key: "item", Value: 1},
				{Key: "date", Value: 1},
			}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$item"},
				{Key: "lastSalesDate", Value: bson.D{{Key: "$last", Value: "$date"}}},
			}}},
		},
	)
}

// TODO: implement TestLastAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $lastN ---

func TestLastNAccumulator_FindLastThreePlayerScoresForSingleGame(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("lastThreeScores", agg.LastNAccumulator(
					[]string{"$playerId", "$score"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "lastThreeScores", Value: bson.D{{Key: "$lastN", Value: bson.D{
					{Key: "input", Value: []string{"$playerId", "$score"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

func TestLastNAccumulator_FindLastThreePlayerScoresAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.LastNAccumulator(
					[]string{"$playerId", "$score"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$lastN", Value: bson.D{
					{Key: "input", Value: []string{"$playerId", "$score"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

func TestLastNAccumulator_UsingSortWithLastN(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.SortStage(agg.Sort("score", agg.Desc)),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.LastNAccumulator(
					[]string{"$playerId", "$score"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$sort", Value: bson.D{{Key: "score", Value: -1}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$lastN", Value: bson.D{
					{Key: "input", Value: []string{"$playerId", "$score"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestLastNAccumulator_ComputeNBasedOnGroupKey
// after $cond operator is implemented

// --- $linearFill ---

// TODO: implement linearFill tests
// after SetWindowFieldsStage is implemented

// --- $locf ---

// TODO: implement locf tests
// after SetWindowFieldsStage is implemented

// --- $max ---

func TestMaxAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$item",
				agg.Accumulate("maxTotalAmount", agg.MaxAccumulator(agg.Multiply("$price", "$quantity"))),
				agg.Accumulate("maxQuantity", agg.MaxAccumulator("$quantity")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$item"},
				{Key: "maxTotalAmount", Value: bson.D{{Key: "$max", Value: bson.D{{Key: "$multiply", Value: bson.A{"$price", "$quantity"}}}}}},
				{Key: "maxQuantity", Value: bson.D{{Key: "$max", Value: "$quantity"}}},
			}}},
		},
	)
}

// TODO: implement TestMaxAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $maxN ---

func TestMaxNAccumulator_FindMaxThreeScoresForSingleGame(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("maxThreeScores", agg.MaxNAccumulator(
					[]string{"$score", "$playerId"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "maxThreeScores", Value: bson.D{{Key: "$maxN", Value: bson.D{
					{Key: "input", Value: []string{"$score", "$playerId"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

func TestMaxNAccumulator_FindMaxThreeScoresAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("maxScores", agg.MaxNAccumulator(
					[]string{"$score", "$playerId"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "maxScores", Value: bson.D{{Key: "$maxN", Value: bson.D{
					{Key: "input", Value: []string{"$score", "$playerId"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestMaxNAccumulator_ComputeNBasedOnGroupKey
// after $cond operator is implemented

// --- $median ---

func TestMedianAccumulator_UseMedianAsAnAccumulator(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				agg.Null,
				agg.Accumulate("test01_median", agg.MedianAccumulator(
					"$test01",
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: nil},
				{Key: "test01_median", Value: bson.D{{Key: "$median", Value: bson.D{
					{Key: "input", Value: "$test01"},
					{Key: "method", Value: "approximate"},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestMedianAccumulator_UseMedianInSetWindowFieldStage
// after $setWindowFields stage is implemented

// --- $mergeObjects ---

func TestMergeObjectsAccumulator_MergeObjectsAsAnAccumulator(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$item",
				agg.Accumulate("mergedSales", agg.MergeObjectsAccumulator(
					"$quantity",
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$item"},
				{Key: "mergedSales", Value: bson.D{{Key: "$mergeObjects", Value: "$quantity"}}},
			}}},
		},
	)
}

// --- $min ---

func TestMinAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$item",
				agg.Accumulate("minQuantity", agg.MinAccumulator("$quantity")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$item"},
				{Key: "minQuantity", Value: bson.D{{Key: "$min", Value: "$quantity"}}},
			}}},
		},
	)
}

// TODO: implement TestMinAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $minMaxScaler ---

// TODO: implement TestMinMaxScalerAccumulator_NormalizeValuesWithCustomRange
// after $setWindowFields stage is implemented

// --- $minN ---

func TestMinNAccumulator_FindMinThreeScoresForSingleGame(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("minScores", agg.MinNAccumulator(
					[]string{"$score", "$playerId"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "minScores", Value: bson.D{{Key: "$minN", Value: bson.D{
					{Key: "input", Value: []string{"$score", "$playerId"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

func TestMinNAccumulator_FindMinThreeDocumentsAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("minScores", agg.MinNAccumulator(
					[]string{"$score", "$playerId"},
					3,
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "minScores", Value: bson.D{{Key: "$minN", Value: bson.D{
					{Key: "input", Value: []string{"$score", "$playerId"}},
					{Key: "n", Value: 3},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestMinNAccumulator_ComputeNBasedOnGroupKey
// after $cond operator is implemented

// --- $percentile ---

func TestPercentileAccumulator_CalculateSingleValueAsAccumulator(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				agg.Null,
				agg.Accumulate("test01_percentiles", agg.PercentileAccumulator(
					"$test01",
					[]float64{0.95},
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: nil},
				{Key: "test01_percentiles", Value: bson.D{{Key: "$percentile", Value: bson.D{
					{Key: "input", Value: "$test01"},
					{Key: "p", Value: []float64{0.95}},
					{Key: "method", Value: "approximate"},
				}}}},
			}}},
		},
	)
}

func TestPercentileAccumulator_CalculateMultipleValuesAsAccumulator(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
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
		},
		bson.A{
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
		},
	)
}

// TODO: implement TestPercentileAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $push ---

func TestPushAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
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
		},
		bson.A{
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
		},
	)
}

// TODO: implement TestPushAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $rank ---

// TODO: implement rank tests
// after $setWindowFields stage is implemented

// --- $setUnion ---

func TestSetUnionAccumulator_FlowersCollection(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$location",
				agg.Accumulate("allFlowers", agg.SetUnionAccumulator("$flowers")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$location"},
				{Key: "allFlowers", Value: bson.D{{Key: "$setUnion", Value: "$flowers"}}},
			}}},
		},
	)
}

// TODO: implement TestSetUnionAccumulator_FlowersCollectionProjection
// after $setUnion expression operator is implemented in operator.go

// --- $shift ---

// TODO: implement TestShiftAccumulator_ShiftUsingPositiveInteger
// after $setWindowFields stage is implemented

// TODO: implement TestShiftAccumulator_ShiftUsingNegativeInteger
// after $setWindowFields stage is implemented

// --- $stdDevPop ---

func TestStdDevPopAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$quiz",
				agg.Accumulate("stdDev", agg.StdDevPopAccumulator("$score")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$quiz"},
				{Key: "stdDev", Value: bson.D{{Key: "$stdDevPop", Value: "$score"}}},
			}}},
		},
	)
}

// TODO: implement TestStdDevPopAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $stdDevSamp ---

// TODO: omits the leading $sample stage (size: 100) as $sample is not yet implemented
func TestStdDevSampAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				agg.Null,
				agg.Accumulate("ageStdDev", agg.StdDevSampAccumulator("$age")),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: nil},
				{Key: "ageStdDev", Value: bson.D{{Key: "$stdDevSamp", Value: "$age"}}},
			}}},
		},
	)
}

// TODO: implement TestStdDevSampAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $sum ---

func TestSumAccumulator_UseInGroupStage(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				bson.D{
					{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
					{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				},
				agg.Accumulate("totalAmount", agg.SumAccumulator(agg.Multiply("$price", "$quantity"))),
				agg.Accumulate("count", agg.SumAccumulator(1)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "day", Value: bson.D{{Key: "$dayOfYear", Value: bson.D{{Key: "date", Value: "$date"}}}}},
					{Key: "year", Value: bson.D{{Key: "$year", Value: bson.D{{Key: "date", Value: "$date"}}}}},
				}},
				{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{"$price", "$quantity"}}}}}},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			}}},
		},
	)
}

// TODO: implement TestSumAccumulator_UseInSetWindowFieldsStage
// after $setWindowFields stage is implemented

// --- $top ---

func TestTopAccumulator_FindTopScore(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.TopAccumulator(
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$top", Value: bson.D{
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: bson.A{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

func TestTopAccumulator_FindTopScoreAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.TopAccumulator(
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$top", Value: bson.D{
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: bson.A{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

// --- $topN ---

func TestTopNAccumulator_FindThreeHighestScores(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.MatchStage(query.Field("gameId", "G1")),
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.TopNAccumulator(
					3,
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "gameId", Value: "G1"}}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$topN", Value: bson.D{
					{Key: "n", Value: 3},
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: bson.A{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

func TestTopNAccumulator_FindThreeHighestScoreDocsAcrossMultipleGames(t *testing.T) {
	assertPipelineEqual(t,
		agg.Pipeline{
			agg.GroupStage(
				"$gameId",
				agg.Accumulate("playerId", agg.TopNAccumulator(
					3,
					[]string{"$playerId", "$score"},
					agg.Sort("score", agg.Desc),
				)),
			),
		},
		bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$gameId"},
				{Key: "playerId", Value: bson.D{{Key: "$topN", Value: bson.D{
					{Key: "n", Value: 3},
					{Key: "sortBy", Value: bson.D{{Key: "score", Value: int32(-1)}}},
					{Key: "output", Value: bson.A{"$playerId", "$score"}},
				}}}},
			}}},
		},
	)
}

// TODO: implement TestTopNAccumulator_ComputingNBasedOnGroupKey
// after $cond and $eq expression operators are implemented
