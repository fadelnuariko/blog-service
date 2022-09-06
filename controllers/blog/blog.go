package blog

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"fadel-blog-services/configs/db"
	"fadel-blog-services/configs/helpers"
	"fadel-blog-services/configs/minio"

	"github.com/gin-gonic/gin"
	miniosdk "github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/bson"
)

var blogColl = db.DB.Collection("blog")

func GetBlogs(c *gin.Context) {
	cursor, err := blogColl.Find(context.Background(), bson.M{})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	blogs := []Blog{}
	if err = cursor.All(db.Ctx, &blogs); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   blogs,
	})
}

func GetBlogBySlug(c *gin.Context) {
	blogSlug := c.Param("slug")
	var blog Blog

	if err := blogColl.FindOne(context.Background(), bson.M{"slug": blogSlug}).Decode(&blog); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   blog,
	})
}

func AddBlog(c *gin.Context) {
	user, _ := c.Get("user")

	var blogData Blog

	if err := c.Bind(&blogData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "can't bind struct",
		})
		c.Abort()
		return
	}

	// get thumbnail and upload it to minio
	fileHeader, err := c.FormFile("image_url")
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

	_, err = minio.MinioClient.PutObject(context.Background(), "fadel-blog", "blog/"+objectName, file, fileHeader.Size, miniosdk.PutObjectOptions{})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// create and uuid string to be stored at db
	newBlogId, err := helpers.CreateUUIDStr()
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	slug := helpers.CreateSlug(blogData.Title)
	newBlog := bson.M{
		"blog_id":    newBlogId,
		"title":      blogData.Title,
		"body":       blogData.Body,
		"image_url":  objectName,
		"image_alt":  blogData.ImageAlt,
		"slug":       slug,
		"published":  "no",
		"created_at": time.Now(),
		"updated_at": time.Now(),
		"updated_by": user.(map[string]string)["user_id"],
	}

	_, err = blogColl.InsertOne(context.Background(), newBlog)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "new blog added successfully",
	})
}

func EditBlogById(c *gin.Context) {
	user, _ := c.Get("user")
	blogId := c.Param("id")
	var blogData Blog

	if err := c.BindJSON(&blogData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "can't bind struct",
		})
		c.Abort()
		return
	}

	// update updated_at, updated_by
	blogData.UpdatedAt = time.Now()
	blogData.UpdatedBy = user.(map[string]string)["user_id"]

	// update process
	pByte, err := bson.Marshal(blogData)
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

	_, err = blogColl.UpdateOne(
		context.Background(),
		bson.M{"blog_id": blogId},
		bson.M{"$set": update},
	)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "blog edited successfully",
	})
}

func DeleteBlogById(c *gin.Context) {
	blogId := c.Param("id")

	// get blog data
	var deletedBlogData Blog
	if err := blogColl.FindOne(context.Background(), bson.M{"blog_id": blogId}).Decode(&deletedBlogData); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// delete blog thumbnail in minio based on image_url (object name)
	err := minio.MinioClient.RemoveObject(context.Background(), "fadel-blog", "blog/"+deletedBlogData.ImageURL, miniosdk.RemoveObjectOptions{})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// delete blog document in mongodb based on id
	_, err = blogColl.DeleteOne(context.Background(), bson.M{"blog_id": blogId})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "blog deleted successfully",
	})
}

func UpdateBlogThumbnail(c *gin.Context) {
	blogId := c.Param("id")
	user, _ := c.Get("user")

	fileHeader, err := c.FormFile("image_url")
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
	info, err := minio.MinioClient.PutObject(context.Background(), "fadel-blog", "blog/"+objectName, file, fileHeader.Size, miniosdk.PutObjectOptions{})
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// 1. get updated blog so we could get the old thumbnail
	// 2. delete the old thumbnail from minio (if exists)

	// 1
	var updatedBlog Blog
	if err := blogColl.FindOne(context.Background(), bson.M{"blog_id": blogId}).Decode(&updatedBlog); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// 2
	blogThumbnail := updatedBlog.ImageURL
	if blogThumbnail != "" {
		err := minio.MinioClient.RemoveObject(context.Background(), "fadel-blog", "blog/"+blogThumbnail, miniosdk.RemoveObjectOptions{})
		if err != nil {
			helpers.SendInternalServerError(c, err)
			return
		}
	}

	// update blog data
	var updatedBlogData Blog

	// add updated_at, updated_by, and image url (thumbnail)
	updatedBlogData.ImageURL = objectName
	updatedBlogData.UpdatedAt = time.Now()
	updatedBlogData.UpdatedBy = user.(map[string]string)["user_id"]

	// update process
	pByte, err := bson.Marshal(updatedBlogData)
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

	_, err = blogColl.UpdateOne(
		context.Background(),
		bson.M{"blog_id": blogId},
		bson.M{"$set": update},
	)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Successfully uploaded new thumbnail of size %d", info.Size),
	})
}

func PublishBlogById(c *gin.Context) {
	blogId := c.Param("id")
	user, _ := c.Get("user")

	var oldBlogData Blog
	var blogData Blog

	if err := blogColl.FindOne(context.Background(), bson.M{"blog_id": blogId}).Decode(&oldBlogData); err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	// add updated_at, updated_by, update published key
	if oldBlogData.Published == "yes" {
		blogData.Published = "no"
	} else {
		blogData.Published = "yes"
	}
	blogData.UpdatedBy = user.(map[string]string)["user_id"]
	blogData.UpdatedAt = time.Now()

	// update process
	pByte, err := bson.Marshal(blogData)
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

	_, err = blogColl.UpdateOne(
		context.Background(),
		bson.M{"blog_id": blogId},
		bson.M{"$set": update},
	)
	if err != nil {
		helpers.SendInternalServerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "blog published/back to draft successfully",
	})
}
