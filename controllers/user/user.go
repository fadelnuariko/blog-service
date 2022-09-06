package user

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"fadel-blog-services/configs/db"
	"fadel-blog-services/configs/helpers"
	"fadel-blog-services/configs/minio"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	miniosdk "github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userColl = db.DB.Collection("user")

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func Login(c *gin.Context) {
	var userData Authentication
	var user User

	if err := c.BindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "can't bind struct",
		})
		c.Abort()
		return
	}

	var error = userColl.FindOne(context.Background(), bson.M{"username": userData.Username}).Decode(&user)
	if error != nil {
		// check if there's an error, maybe due to wrong username
		c.JSON(http.StatusOK, gin.H{
			"status":  "failed",
			"message": "Invalid username or password",
		})
		c.Abort()
		return
	}

	if !CheckPasswordHash(userData.Password, user.Password) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "failed",
			"message": "Invalid username or password",
		})
		c.Abort()
		return
	}

	sign := jwt.New(jwt.GetSigningMethod("HS256"))
	claims := sign.Claims.(jwt.MapClaims)
	claims["user_id"] = user.UserId
	claims["username"] = user.Username

	token, err := sign.SignedString([]byte("secret"))
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":                  user.UserId,
			"profile_picture_url": user.ProfilePictureURL,
			"username":            userData.Username,
			"token":               token,
		},
	})
}

func Auth(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod("HS256") != token.Method {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte("secret"), nil
	})

	if token != nil && err == nil {
		fmt.Println("token verified")
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Not authorized",
			"error":   err.Error(),
		})
		c.Abort()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok && !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "not authorized",
			"error":   err.Error(),
		})
		c.Abort()
		return
	}

	c.Set("user", map[string]string{
		"user_id":  claims["user_id"].(string),
		"username": claims["username"].(string),
	})
	c.Next()
}

func GetUsers(c *gin.Context) {
	cursor, err := userColl.Find(context.Background(), bson.M{})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	var users []User
	if err = cursor.All(db.Ctx, &users); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// delete password from struct by pass an empty string, for security purpose
	for k := range users {
		users[k].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   users,
	})
}

func GetUserById(c *gin.Context) {
	id := c.Param("id")
	var user User

	if err := userColl.FindOne(context.Background(), bson.M{"user_id": id}).Decode(&user); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// delete password from struct by pass an empty string, for security purpose
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   user,
	})
}

func AddUser(c *gin.Context) {
	user, _ := c.Get("user")
	var userData User

	if err := c.BindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "can't bind struct",
		})
		c.Abort()
		return
	}

	// check if there's user with the same username
	var checkUsername User
	err := userColl.FindOne(context.Background(), bson.M{"username": userData.Username}).Decode(&checkUsername)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			helpers.SendInternalServerError(c, err)
			return
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status":  "failed",
			"message": fmt.Sprintf("Username %v sudah terpakai", userData.Username),
		})
		c.Abort()
		return
	}

	// create and uuid string to be stored at db
	newUserId, err := helpers.CreateUUIDStr()
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.MinCost)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	newUser := bson.M{
		"user_id":             newUserId,
		"full_name":           userData.Fullname,
		"username":            userData.Username,
		"password":            string(hash),
		"profile_picture_url": "", // default is empty
		"phone_number":        userData.PhoneNumber,
		"created_at":          time.Now(),
		"updated_at":          time.Now(),
		"updated_by":          user.(map[string]string)["user_id"],
	}

	_, err = userColl.InsertOne(context.Background(), newUser)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "new user added successfully",
	})
}

func EditUserById(c *gin.Context) {
	user, _ := c.Get("user")
	id := c.Param("id")
	var userData User

	if err := c.BindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "can't bind struct",
		})
		c.Abort()
		return
	}

	// add updated_at, updated_by
	userData.UpdatedAt = time.Now()
	userData.UpdatedBy = user.(map[string]string)["user_id"]

	// update process
	pByte, err := bson.Marshal(userData)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	var update bson.M
	err = bson.Unmarshal(pByte, &update)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	_, err = userColl.UpdateOne(
		context.Background(),
		bson.M{"user_id": id},
		bson.M{"$set": update},
	)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "user edited successfully",
	})
}

func DeleteUserById(c *gin.Context) {
	userId := c.Param("id")

	// get user data
	var deletedUserData User
	if err := userColl.FindOne(context.Background(), bson.M{"user_id": userId}).Decode(&deletedUserData); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// delete user profile picture in minio based on image_url (object name)
	userProfilePicture := deletedUserData.ProfilePictureURL
	if userProfilePicture != "" {
		err := minio.MinioClient.RemoveObject(context.Background(), "fadel-blog", "profile/"+userProfilePicture, miniosdk.RemoveObjectOptions{})
		if err != nil {
			helpers.SendInternalServerError(c, err)
			return
		}
	}

	// delete user document in mongodb based on id
	_, err := userColl.DeleteOne(context.Background(), bson.M{"user_id": userId})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "user deleted successfully",
	})
}

func UpdateProfileImage(c *gin.Context) {
	user, _ := c.Get("user")

	fileHeader, err := c.FormFile("profile_image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": err.Error(),
		})
		c.Abort()
		return
	}

	// validate image type
	err = helpers.ValidateImage(c, fileHeader.Filename)
	if err != nil {
		return
	}

	// open file from file header
	file, err := fileHeader.Open()
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// upload to minio
	objectName := fmt.Sprint(time.Now().UnixNano()) + filepath.Ext(fileHeader.Filename)
	info, err := minio.MinioClient.PutObject(context.Background(), "fadel-blog", "profile/"+objectName, file, fileHeader.Size, miniosdk.PutObjectOptions{})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// 1. get updated user so we could get the old profile image
	// 2. delete the old profile image from minio (if exists)

	// 1
	var updatedUser User
	if err := userColl.FindOne(context.Background(), bson.M{"user_id": user.(map[string]string)["user_id"]}).Decode(&updatedUser); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// 2
	userProfilePicture := updatedUser.ProfilePictureURL
	if userProfilePicture != "" {
		err := minio.MinioClient.RemoveObject(context.Background(), "fadel-blog", "profile/"+userProfilePicture, miniosdk.RemoveObjectOptions{})
		if err != nil {
			helpers.SendInternalServerError(c, err)
			return
		}
	}

	// update user data
	var updatedUserData User

	// add updated_at, updated_by, and profile image url
	updatedUserData.ProfilePictureURL = objectName
	updatedUserData.UpdatedAt = time.Now()
	updatedUserData.UpdatedBy = user.(map[string]string)["user_id"]

	// update process
	pByte, err := bson.Marshal(updatedUserData)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	var update bson.M
	err = bson.Unmarshal(pByte, &update)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	_, err = userColl.UpdateOne(
		context.Background(),
		bson.M{"user_id": user.(map[string]string)["user_id"]},
		bson.M{"$set": update},
	)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data": gin.H{
			"profile_picture_url": updatedUserData.ProfilePictureURL,
		},
		"message": fmt.Sprintf("Successfully uploaded new profile image of size %d", info.Size),
	})
}
