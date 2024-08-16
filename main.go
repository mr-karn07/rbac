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

	// Periodically reload policies
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

	// API to create a new resource and assign the creator as admin
	app.Post("/resource", func(c *fiber.Ctx) error {
		resource := c.FormValue("resource")
		user := c.FormValue("user")
		// Assign the user as admin of the new resource
		_, err := enforcer.AddPolicy(user, resource, "admin")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to assign admin role"})
		}
		return c.JSON(fiber.Map{"message": "Resource created and admin role assigned"})
	})

	// API to assign a role to a user for a resource
	app.Post("/resource/:resource/assign", func(c *fiber.Ctx) error {
		resource := c.Params("resource")
		user := c.FormValue("user")
		role := c.FormValue("role")
		// Assign the specified role to the user for the resource
		_, err := enforcer.AddPolicy(user, resource, role)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to assign role"})
		}
		return c.JSON(fiber.Map{"message": "Role assigned successfully"})
	})

	app.Use(enforcerMiddleware.Middleware)

	// Sample resource access APIs
	app.Get("/resource/view", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the view handler")
		return c.JSON(fiber.Map{"message": "Resource viewed"})
	})

	app.Post("/resource/edit", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the edit handler")
		return c.JSON(fiber.Map{"message": "Resource edited"})
	})

	app.Delete("/resource/delete", func(c *fiber.Ctx) error {
		fmt.Println("hi, i am back here in the delete handler")
		return c.JSON(fiber.Map{"message": "Resource deleted"})
	})

	log.Fatal(app.Listen(":3000"))
}
