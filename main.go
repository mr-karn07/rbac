// This main package shows the usage of rbac package
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mr-karn07/rbac.git/auth"
	"github.com/mr-karn07/rbac.git/config"
	"github.com/mr-karn07/rbac.git/opensearch"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.LoadConfig()

	adapter, err := opensearch.NewAdapter(cfg.OpenSearchAddresses, cfg.Index)
	if err != nil {
		log.Fatalf("Error creating OpenSearch adapter: %v", err)
	}

	enforcer, err := casbin.NewEnforcer(cfg.ModelPath, adapter)
	if err != nil {
		log.Fatalf("Error initializing Casbin enforcer: %v", err)
	}

	enforcerMiddleware := auth.NewEnforcerMiddleware(enforcer)

	// Periodically reload policies to load all newly added policies
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Can be adjusted as per requirement
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Reloading policies from OpenSearch...")
			err := enforcer.LoadPolicy()
			if err != nil {
				log.Printf("Error reloading policies: %v", err)
			} else {
				log.Println("Policies reloaded successfully")
			}
		}
	}()

	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		log.Printf("API hit: %s %s", c.Method(), c.Path())
		return c.Next()
	})

	// API to assign a role to a user for a resource along with action
	app.Post("/resource", func(c *fiber.Ctx) error {
		req := struct {
			Resource string `json:"resource"`
			User     string `json:"user"`
			Role     string `json:"role"`
			Action   string `json:"action"`
		}{}
		err := c.BodyParser(&req)
		if err != nil {
			return err
		}
		// Assign the specified role to the user for the resource
		_, err = enforcer.AddPolicy(req.User, req.Role, req.Resource, req.Action)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to assign role"})
		}
		return c.JSON(fiber.Map{"message": "Resouce created and Role assigned successfully"})
	})

	app.Use(enforcerMiddleware.Middleware)

	// Sample resource access APIs
	app.Post("/datascience/create-pipeline", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the /datascience/create-pipeline handler")
		return c.JSON(fiber.Map{"message": "pipeline created"})
	})

	app.Get("/datascience/get-pipeline", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the /datascience/get-pipeline handler")
		return c.JSON(fiber.Map{"message": "pipeline viewed"})
	})

	app.Delete("/datascience/delete-pipeline", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the /datascience/delete-pipeline handler")
		return c.JSON(fiber.Map{"message": "pipeline deleted"})
	})

	app.Get("/developer", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the /developer handler")
		return c.JSON(fiber.Map{"message": "developer fetched"})
	})

	app.Get("/core", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the /core handler")
		return c.JSON(fiber.Map{"message": "core fetched"})
	})

	app.Get("/infra", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the /infra handler ")
		return c.JSON(fiber.Map{"message": "infra fetched"})
	})

	log.Fatal(app.Listen(":3000"))
}
