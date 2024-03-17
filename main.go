package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const mongoURI = 'add'

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name,omitempty"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

/* Connect with the database! */
func ConnectToDB() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

	defer cancel()

	_, err = mongo.Connect(ctx)

	if err != nil {
		log.Fatal(err)
		return err
	}

	fmt.Println("MongoDB connection successfully!")

	db := client.Database("fiber-hrms")
	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil
}

func main() {
	err := ConnectToDB()
	if err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	/* Get all employees and also handling errors! */
	app.Get("/employee", func(c *fiber.Ctx) error {

		query := bson.D{{}}
		var employees []Employee = make([]Employee, 0)

		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			c.Status(500).SendString(err.Error())
		}
		err = cursor.All(c.Context(), &employees)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)
	})

	/* Post new employee && return createdRecord! */
	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")

		var employee Employee

		err := c.BodyParser(employee)

		if err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = primitive.NewObjectID().String()

		insertedResult, err := collection.InsertOne(c.Context(), employee)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertedResult.InsertedID}}

		var createdRecord Employee
		collection.FindOne(c.Context(), filter).Decode(&createdRecord)

		return c.Status(201).JSON(createdRecord)

	})

	/* update employee && returns updated employee */
	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		employeeId, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			return c.SendStatus(400)
		}

		var employee Employee

		err = c.BodyParser(employee)

		if err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeId}}

		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam

		return c.Status(200).JSON(employee)
	})

	/* Delete employee && returns success response*/
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {

		employeeId, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeId}}

		resulted, err := mg.Db.Collection("employees").DeleteOne(c.Context(), query)

		if err != nil {
			return c.SendStatus(504)
		}

		if resulted.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).SendString("Deleted Employee")
	})

	/* Starting server on port:3000*/
	log.Fatal(app.Listen(":3000"))
}
