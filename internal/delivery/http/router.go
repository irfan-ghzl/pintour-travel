package httpdelivery

import (
	"net/http"

	_ "github.com/irfan-ghzl/pintour-travel/docs"
	bookingsvc "github.com/irfan-ghzl/pintour-travel/internal/application/booking"
	inquirysvc "github.com/irfan-ghzl/pintour-travel/internal/application/inquiry"
	quotationsvc "github.com/irfan-ghzl/pintour-travel/internal/application/quotation"
	toursvc "github.com/irfan-ghzl/pintour-travel/internal/application/tour"
	usersvc "github.com/irfan-ghzl/pintour-travel/internal/application/user"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Services bundles all application services needed to register routes.
type Services struct {
	Tour      *toursvc.TourService
	Booking   *bookingsvc.BookingService
	Inquiry   *inquirysvc.InquiryService
	Quotation *quotationsvc.QuotationService
	User      *usersvc.UserService
	JWTSecret string
}

// RegisterRoutes mounts all API routes onto the Echo instance.
func RegisterRoutes(e *echo.Echo, svc Services) {
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "version": "1.0.0"})
	})

	tourH := NewTourHandler(svc.Tour)
	bookingH := NewBookingHandler(svc.Booking)
	inquiryH := NewInquiryHandler(svc.Inquiry)
	quotationH := NewQuotationHandler(svc.Quotation)
	userH := NewUserHandler(svc.User)
	dashH := NewDashboardHandler()

	api := e.Group("/api/v1")

	// ── Public routes ──────────────────────────────────────────────────────────
	api.POST("/auth/login", userH.Login)

	api.GET("/packages", tourH.ListPackages)
	api.GET("/packages/:slug", tourH.GetPackage)
	api.GET("/packages/:package_id/gallery", tourH.ListGallery)

	api.GET("/destinations", tourH.ListDestinations)
	api.GET("/testimonials", tourH.ListTestimonials)

	api.POST("/inquiries", inquiryH.CreateInquiry)

	// Quotation print is public (UUID is effectively unguessable)
	api.GET("/quotations/:id/print", quotationH.PrintQuotation)

	// ── Protected admin routes ─────────────────────────────────────────────────
	jwtMW := JWTMiddleware(svc.JWTSecret)
	admin := api.Group("/admin", jwtMW)

	admin.GET("/auth/me", userH.Me)
	admin.GET("/dashboard/stats", dashH.GetStats)

	// Packages
	admin.POST("/packages", tourH.CreatePackage)
	admin.PUT("/packages/:id", tourH.UpdatePackage)
	admin.DELETE("/packages/:id", tourH.DeletePackage)

	// Gallery
	admin.POST("/packages/:package_id/gallery", tourH.AddGalleryImage)
	admin.DELETE("/packages/:package_id/gallery/:image_id", tourH.DeleteGalleryImage)

	// Itinerary
	admin.GET("/packages/:package_id/itinerary", tourH.ListItinerary)
	admin.POST("/packages/:package_id/itinerary", tourH.AddItineraryItem)
	admin.PUT("/packages/:package_id/itinerary/:item_id", tourH.UpdateItineraryItem)
	admin.DELETE("/packages/:package_id/itinerary/:item_id", tourH.DeleteItineraryItem)

	// Inquiries
	admin.GET("/inquiries", inquiryH.ListInquiries)
	admin.PATCH("/inquiries/:id/status", inquiryH.UpdateInquiryStatus)

	// Quotations
	admin.POST("/quotations", quotationH.CreateQuotation)
	admin.GET("/quotations", quotationH.ListQuotations)
	admin.GET("/quotations/:id", quotationH.GetQuotation)
	admin.GET("/quotations/:id/print", quotationH.PrintQuotation)

	// Bookings / Manifest
	admin.GET("/bookings", bookingH.ListBookings)
	admin.GET("/bookings/:id", bookingH.GetBooking)
	admin.POST("/bookings", bookingH.CreateBooking)
	admin.PATCH("/bookings/:id/payment-status", bookingH.UpdatePaymentStatus)
	admin.PATCH("/bookings/:id/booking-status", bookingH.UpdateBookingStatus)
	admin.DELETE("/bookings/:id", bookingH.DeleteBooking)
}
