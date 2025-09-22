package main

import "time"

type contenedor struct {
	Image string `json:"image" bson:"image" validate:"required"`
}

type registerUser struct {
	Email    string `json:"email" bson:"email" validate:"required,email"`
	Password string `json:"password" bson:"password" validate:"required"`
	Name     string `json:"name" bson:"name" validate:"required"`
}

type loginUser struct {
	Email    string `json:"email" bson:"email" validate:"required,email"`
	Password string `json:"password" bson:"password" validate:"required"`
}

type loginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type verifyTokenResponse struct {
	Valid bool `json:"valid"`
	User  struct {
		Sub       string `json:"sub"`
		Email     string `json:"email"`
		DbName    string `json:"dbName"`
		Role      string `json:"role"`
		SessionId string `json:"sessionId"`
	} `json:"user"`
}

type ContainerRecord struct {
	UserID        string    `bson:"userId" json:"userId"`
	ContainerName string    `bson:"containerName" json:"containerName"`
	Status        bool      `bson:"status" json:"status"`
	CreatedAt     time.Time `bson:"createdAt" json:"createdAt"`
}
