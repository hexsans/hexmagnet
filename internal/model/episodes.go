package model

type Episodes map[int]map[int]struct{}

func (e Episodes) AddEpisode(season, episode int) Episodes {
	_, epMapOk := e[season]
	if !epMapOk {
		e[season] = make(map[int]struct{})
	}

	e[season][episode] = struct{}{}

	return e
}

func (e Episodes) AddSeason(season int) Episodes {
	e[season] = make(map[int]struct{})
	return e
}

type Season struct {
	Season   int
	Episodes []int `json:"omitempty"`
}
