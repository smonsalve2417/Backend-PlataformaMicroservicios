package main

type newDocker struct {
	Id    string `json:"id" bson:"id"`
	Image string `json:"image" bson:"image" validate:"required"`
}
