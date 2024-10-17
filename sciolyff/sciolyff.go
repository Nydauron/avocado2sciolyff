package sciolyff

type SciolyFF struct {
	Tournament TournamentMetadata `yaml:"Tournament"`
	Tracks     []Track            `yaml:"Tracks,omitempty"`
	Events     []Event            `yaml:"Events"`
	Teams      []School           `yaml:"Teams"`
	Placings   []Placing          `yaml:"Placings"`
}

type Track struct {
	Name string `yaml:"name"`
}

type TournamentMetadata struct {
	Name      string `yaml:"name"`
	ShortName string `yaml:"short name,omitempty"`
	Location  string `yaml:"location"`
	Level     string `yaml:"level"`
	State     string `yaml:"state"`
	Division  string `yaml:"division"`
	Year      int    `yaml:"year"`
	Date      string `yaml:"date"`
}

type Event struct {
	Name               string `yaml:"name"`
	IsTrial            bool   `yaml:"trial"`
	TrialedNormalEvent bool   `yaml:"trialed"`
	ScoringObjective   string `yaml:"scoring,omitempty"`
}

type Placing struct {
	Event        string `yaml:"event"`
	TeamNumber   uint   `yaml:"team"`
	Participated bool   `yaml:"participated"`
	EventDQ      bool   `yaml:"disqualified"`
	Exempt       bool   `yaml:"exempt"`
	Unknown      bool   `yaml:"unknown"`
	Tie          bool   `yaml:"tie"`
	Place        uint   `yaml:"place"`
	TrackPlace   uint   `yaml:"track place,omitempty"`
}

type School struct {
	TeamNumber uint   `yaml:"number"`
	Name       string `yaml:"school"`
	Track      string `yaml:"track"`
	Scores     []uint `yaml:"-"`
	TotalScore string `yaml:"-"`
	Rank       string `yaml:"-"`
}
