package spotify

import "google.golang.org/adk/tool"

type SkipfWrapperResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipfWrapper(ctx tool.Context, _ struct{}) (SkipfWrapperResult, error) {
	if err := c.skipForward(); err != nil {
		return SkipfWrapperResult{
			Success: false,
		}, err
	}

	return SkipfWrapperResult{
		Success: true,
	}, nil
}

// type getChuckNorrisJokeResult struct {
// 	Joke string		`json:"joke"`
// }

// func getChuckNorrisJoke(ctx tool.Context, _ struct{}) (getChuckNorrisJokeResult, error) {
// 	return getChuckNorrisJokeResult{
// 		Joke: "Why did Chuck Norris cross the road? Because he was badass.",
// 	}, nil
// }
