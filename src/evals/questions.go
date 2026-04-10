package main

// Question represents a single eval question targeting a specific genre.
type Question struct {
	ID       string
	Text     string
	Genre    string // the genre to ask about (human-readable)
	GenreURI string // full URI for reference
	HasData  bool   // whether we expect data in genre.ttl for this question
}

// AllQuestions returns the 9 evaluation questions.
// Each question is asked about a specific genre to ground the eval.
func AllQuestions() []Question {
	return []Question{
		{
			ID:       "q1_popular_where",
			Text:     "Where is Reggaeton most popular?",
			Genre:    "Reggaeton",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#reggaeton",
			HasData:  true,
		},
		{
			ID:       "q2_listen_locations",
			Text:     "Where can I go to listen to K-Pop? What locations or platforms?",
			Genre:    "K-Pop",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#k-pop",
			HasData:  true,
		},
		{
			ID:       "q3_top_songs",
			Text:     "What are the top songs in K-Pop?",
			Genre:    "K-Pop",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#k-pop",
			HasData:  true,
		},
		{
			ID:       "q4_demographics",
			Text:     "What are the main audience demographics of people who listen to Reggaeton?",
			Genre:    "Reggaeton",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#reggaeton",
			HasData:  false, // no demographic data in genre.ttl
		},
		{
			ID:       "q5_artists",
			Text:     "Who are the main artists affiliated with Rock Music?",
			Genre:    "Rock Music",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#rock-music",
			HasData:  true,
		},
		{
			ID:       "q6_characteristics",
			Text:     "What are the main characteristics of Reggaeton? What language, tempo, and time period is it associated with?",
			Genre:    "Reggaeton",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#reggaeton",
			HasData:  true,
		},
		{
			ID:       "q7_instruments",
			Text:     "What types of instruments are used in Classic Rock songs?",
			Genre:    "Classic Rock",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#classic-rock",
			HasData:  true,
		},
		{
			ID:       "q8_cultural_moments",
			Text:     "What are defining cultural moments associated with Reggaeton?",
			Genre:    "Reggaeton",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#reggaeton",
			HasData:  false, // no cultural moments in genre.ttl (only in merged file)
		},
		{
			ID:       "q9_related_genres",
			Text:     "What genres are related or similar to Pop Music? What are its subgenres and parent genres?",
			Genre:    "Pop Music",
			GenreURI: "http://thekgguys.bootcamp.ai/genres#pop-music",
			HasData:  true,
		},
	}
}
