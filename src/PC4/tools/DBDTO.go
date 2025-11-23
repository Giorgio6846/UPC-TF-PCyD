package tools

import "go.mongodb.org/mongo-driver/bson/primitive"

type Rating struct {
	UserID  int32   `bson:"userId"`
	MovieID int32   `bson:"movieId"`
	Rating  float64 `bson:"rating"`
}

type Movie struct {
	ID     int32    `bson:"_id"`
	IMDB   int32    `bson:"imdb,omitempty"`
	TMDB   int32    `bson:"tmdb,omitempty"`
	Title  string   `bson:"title"`
	Genres []string `bson:"genres,omitempty"`
}

type User struct {
	ID      int                  `bson:"_id"`
	Ratings []primitive.ObjectID `bson:"ratingIds,omitempty"`
	Tags    []primitive.ObjectID `bson:"tagIds,omitempty"`
}
type Ratings struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	UserID  int32              `bson:"userId"`
	MovieID int32              `bson:"movieId"`
	Rating  float64            `bson:"rating"`
}

type Tags struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	UserID  int32              `bson:"userId"`
	MovieID int32              `bson:"movieId"`
	Tag     string             `bson:"tag"`
}

type LinksParsed struct {
	MovieId int
	Imdb    int32
	Tmdb    int32
}

type UserInfo struct {
	Email    string
	Password string
}

type UserVector map[int]float64

type SimilarityVector struct {
	TargetVector UserVector
	UsersVector  map[int]UserVector
}

type Recommended struct {
	MovieID int
	Score   float64
	Count   int
}
