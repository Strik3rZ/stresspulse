package logs

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
)

type FakeLogGenerator struct {
	logType    string
	interval   time.Duration
	enabled    bool
	ctx        context.Context
	logger     *log.Logger
}

func NewFakeLogGenerator(logType string, interval time.Duration, logger *log.Logger) *FakeLogGenerator {
	return &FakeLogGenerator{
		logType:  logType,
		interval: interval,
		enabled:  false,
		logger:   logger,
	}
}

func (flg *FakeLogGenerator) Start(ctx context.Context) {
	if flg.enabled {
		return
	}
	
	flg.enabled = true
	flg.ctx = ctx
	go flg.generateLogs()
}

func (flg *FakeLogGenerator) Stop() {
	if !flg.enabled {
		return
	}
	
	flg.enabled = false
}

func (flg *FakeLogGenerator) generateLogs() {
	ticker := time.NewTicker(flg.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-flg.ctx.Done():
			return
		case <-ticker.C:
			flg.generateLogEntry()
		}
	}
}

func (flg *FakeLogGenerator) generateLogEntry() {
	switch flg.logType {
	case "java":
		flg.generateJavaLog()
	case "web":
		flg.generateWebLog()
	case "microservice":
		flg.generateMicroserviceLog()
	case "database":
		flg.generateDatabaseLog()
	case "ecommerce":
		flg.generateEcommerceLog()
	default:
		flg.generateGenericLog()
	}
}

func (flg *FakeLogGenerator) generateJavaLog() {
	patterns := []func(){
		func() {
			classes := []string{
				"com.example.service.UserService",
				"com.example.controller.PaymentController", 
				"com.example.repository.OrderRepository",
				"com.example.config.DatabaseConfig",
				"com.example.util.CacheManager",
				"org.springframework.boot.SpringApplication",
				"com.example.security.AuthenticationService",
			}
			
			messages := []string{
				"Processing user authentication request",
				"Cache miss for key: user_session_%d",
				"Database connection established successfully",
				"Starting transaction for order processing",
				"Validation completed for request ID: %s",
				"Memory usage: %d%% of available heap",
				"GC collection completed in %dms",
			}
			
			class := classes[rand.Intn(len(classes))]
			message := messages[rand.Intn(len(messages))]
			
			switch rand.Intn(4) {
			case 0:
				flg.logger.Printf("INFO  [main] %s - %s", class, fmt.Sprintf(message, rand.Intn(10000), generateRandomString(8), rand.Intn(100), rand.Intn(500)))
			case 1: 
				flg.logger.Printf("DEBUG [http-thread-%d] %s - %s", rand.Intn(20), class, fmt.Sprintf(message, rand.Intn(10000), generateRandomString(8), rand.Intn(100), rand.Intn(500)))
			case 2:
				flg.logger.Printf("WARN  [scheduler-1] %s - Connection pool running low: %d connections available", class, rand.Intn(5)+1)
			case 3:
				flg.logger.Printf("ERROR [main] %s - Failed to process request: %s", class, generateRandomError())
			}
		},
		func() {
			flg.logger.Printf("INFO  [main] o.s.b.w.embedded.tomcat.TomcatWebServer - Tomcat started on port(s): 8080 (http)")
		},
		func() {
			queries := []string{
				"SELECT * FROM users WHERE id = %d",
				"UPDATE orders SET status = 'COMPLETED' WHERE id = %d", 
				"INSERT INTO audit_log (action, user_id, timestamp) VALUES ('%s', %d, NOW())",
				"DELETE FROM sessions WHERE expires_at < NOW()",
			}
			query := queries[rand.Intn(len(queries))]
			flg.logger.Printf("DEBUG [HikariPool-1] org.hibernate.SQL - %s", fmt.Sprintf(query, rand.Intn(1000), generateRandomString(6)))
		},
	}
	
	patterns[rand.Intn(len(patterns))]()
}

func (flg *FakeLogGenerator) generateWebLog() {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	endpoints := []string{
		"/api/users",
		"/api/orders", 
		"/api/products",
		"/api/auth/login",
		"/api/payments",
		"/health",
		"/metrics",
		"/api/search",
	}
	
	statuses := []int{200, 201, 400, 401, 403, 404, 500, 503}
	
	method := methods[rand.Intn(len(methods))]
	endpoint := endpoints[rand.Intn(len(endpoints))]
	status := statuses[rand.Intn(len(statuses))]
	responseTime := rand.Intn(2000) + 10
	ip := fmt.Sprintf("192.168.1.%d", rand.Intn(254)+1)
	
	flg.logger.Printf("%s - - [%s] \"%s %s HTTP/1.1\" %d %d \"-\" \"Mozilla/5.0\" %dms",
		ip,
		time.Now().Format("02/Jan/2006:15:04:05 -0700"),
		method,
		endpoint,
		status,
		rand.Intn(50000)+100,
		responseTime)
}

func (flg *FakeLogGenerator) generateMicroserviceLog() {
	services := []string{
		"user-service",
		"order-service", 
		"payment-service",
		"notification-service",
		"inventory-service",
		"api-gateway",
	}
	
	service := services[rand.Intn(len(services))]
	traceId := generateRandomString(16)
	spanId := generateRandomString(8)
	
	patterns := []func(){
		func() {
			flg.logger.Printf("INFO  [%s] [trace=%s,span=%s] Processing request for user: %d", 
				service, traceId, spanId, rand.Intn(10000))
		},
		func() {
			flg.logger.Printf("DEBUG [%s] [trace=%s,span=%s] Circuit breaker state: CLOSED", 
				service, traceId, spanId)
		},
		func() {
			flg.logger.Printf("WARN  [%s] [trace=%s,span=%s] Rate limit approaching: %d requests/minute", 
				service, traceId, spanId, rand.Intn(1000)+800)
		},
		func() {
			flg.logger.Printf("ERROR [%s] [trace=%s,span=%s] Service unavailable: %s", 
				service, traceId, spanId, generateRandomError())
		},
	}
	
	patterns[rand.Intn(len(patterns))]()
}

func (flg *FakeLogGenerator) generateDatabaseLog() {
	operations := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE INDEX", "VACUUM",
	}
	
	tables := []string{
		"users", "orders", "products", "payments", "inventory", "audit_log",
	}
	
	operation := operations[rand.Intn(len(operations))]
	table := tables[rand.Intn(len(tables))]
	duration := rand.Intn(5000) + 1
	
	flg.logger.Printf("LOG:  duration: %d.%03d ms  statement: %s operation on table %s affected %d rows",
		duration/1000, duration%1000, operation, table, rand.Intn(100)+1)
}

func (flg *FakeLogGenerator) generateEcommerceLog() {
	events := []string{
		"USER_LOGIN", "PRODUCT_VIEW", "CART_ADD", "CHECKOUT_START", 
		"PAYMENT_SUCCESS", "ORDER_PLACED", "SHIPPING_LABEL_CREATED",
	}
	
	event := events[rand.Intn(len(events))]
	userId := rand.Intn(10000) + 1
	sessionId := generateRandomString(32)
	
	flg.logger.Printf("INFO  [event-processor] Event: %s | UserId: %d | SessionId: %s | Amount: $%.2f",
		event, userId, sessionId, float64(rand.Intn(50000))/100.0)
}

func (flg *FakeLogGenerator) generateGenericLog() {
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG"}
	components := []string{"auth", "db", "cache", "queue", "scheduler", "monitor"}
	
	level := levels[rand.Intn(len(levels))]
	component := components[rand.Intn(len(components))]
	
	messages := []string{
		"Operation completed successfully",
		"Processing batch of %d items", 
		"Configuration reloaded",
		"Health check passed",
		"Cache eviction completed",
		"Timeout waiting for response",
	}
	
	message := messages[rand.Intn(len(messages))]
	flg.logger.Printf("%s  [%s] %s", level, component, fmt.Sprintf(message, rand.Intn(1000)+1))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomError() string {
	errors := []string{
		"Connection timeout after 30s",
		"Invalid JSON format in request body",
		"Database constraint violation", 
		"Authentication token expired",
		"Rate limit exceeded",
		"Service temporarily unavailable",
		"Invalid parameter: expected number, got string",
		"Memory allocation failed",
	}
	return errors[rand.Intn(len(errors))]
} 
