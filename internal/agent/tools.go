package agent

import "google.golang.org/adk/tool"

type getChuckNorrisJokeResult struct {
	Joke string `json:"joke"`
}

func getChuckNorrisJoke(ctx tool.Context, _ struct{}) (getChuckNorrisJokeResult, error) {
	return getChuckNorrisJokeResult{
		Joke: "Why did Chuck Norris cross the road? Because he was badass.",
	}, nil
}
