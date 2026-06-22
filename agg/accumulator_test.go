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
	initArgs := agg.Array([]any{"$city", "Bettles"})
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
						&initArgs,
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
