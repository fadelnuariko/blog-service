package blog

import "time"

type Blog struct {
	BlogId    string    `bson:"blog_id,omitempty" json:"blog_id"`
	Title     string    `form:"title" bson:"title,omitempty" json:"title"`
	Body      string    `form:"body" bson:"body,omitempty" json:"body"`
	ImageURL  string    `bson:"image_url,omitempty" json:"image_url"`
	ImageAlt  string    `form:"image_alt" bson:"image_alt" json:"image_alt"`
	Slug      string    `bson:"slug,omitempty" json:"slug"`
	Published string    `bson:"published,omitempty" json:"published"`
	CreatedAt time.Time `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at,omitempty" json:"updated_at"`
	UpdatedBy string    `bson:"updated_by,omitempty" json:"updated_by"`
}
