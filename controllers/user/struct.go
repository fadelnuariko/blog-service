package user

import "time"

type Authentication struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type User struct {
	UserId            string    `bson:"user_id,omitempty" json:"user_id"`
	Fullname          string    `bson:"full_name,omitempty" json:"full_name"`
	Username          string    `bson:"username,omitempty" json:"username"`
	Password          string    `bson:"password,omitempty" json:"password,omitempty"`
	ProfilePictureURL string    `bson:"profile_picture_url,omitempty" json:"profile_picture_url"`
	PhoneNumber       string    `bson:"phone_number,omitempty" json:"phone_number"`
	CreatedAt         time.Time `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt         time.Time `bson:"updated_at,omitempty" json:"updated_at"`
	UpdatedBy         string    `bson:"updated_by,omitempty" json:"updated_by"`
}
