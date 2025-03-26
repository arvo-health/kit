package kit

//
//import (
//	"context"
//	"net/http"
//	"regexp"
//
//	userApp "github.com/arvo-health/arvo-hub-back/app/user_app/client"
//	"github.com/gofiber/fiber/v2"
//	"github.com/gofiber/fiber/v2/middleware/skip"
//)
//
//type Authenticator interface {
//	Authenticate(ctx context.Context, in userApp.AuthenticateIn) (userApp.AuthenticateOut, error)
//}
//
//// Add UserApp authentication middleware to all routes except POST - v1/claims.
//// The UserAuthMiddleware performs 3 tasks:
////
//// 1. Authenticate the user via UserApp
//// 2. Set the permissions object in the context so the handlers can use it for authorization purposes
//// 3. Set the user email in the context so the handlers can use it for filtering purposes
//func UserAuthMiddleware(client *userApp.UserAppClient) Middleware {
//	userAuthHandler := func(ctx *fiber.Ctx) error {
//		in := userApp.AuthenticateIn{Email: ctx.Get("email"), AccessToken: ctx.Get("access-token")}
//
//		ctx.Locals(kit.KeyUserEmail, in.Email)
//
//		// TODO: Add user role to the client.Authenticate() response
//		res, err := client.Authenticate(ctx.Context(), in)
//		if err != nil {
//			return kit.ErrUnauthorized.WrapCause(err)
//		}
//
//		// Set Permissions in the context so the handler can access it.
//		ctx.Context().SetUserValue(userApp.PermissionsCtxKey, res.Permissons)
//		// Set Email in the context so the handler can access it.
//		ctx.Context().SetUserValue(userApp.EmailCtxKey, in.Email)
//
//		// TODO: Add user role to the Locals so the handler can access it.
//		ctx.Locals(kit.KeyUserCompany, res.Company)
//		ctx.Locals(kit.KeyUserCompanyCategory, res.CompanyCategory)
//		ctx.Locals(kit.KeyUserPermissions, res.Permissons)
//		return ctx.Next()
//	}
//
//	return skip.New(userAuthHandler, func(ctx *fiber.Ctx) bool {
//		path := string(ctx.Request().URI().Path())
//		method := string(ctx.Request().Header.Method())
//
//		switch {
//		// Skip access_token auth on health check route.
//		case path == "/v1/health":
//			return true
//		// Skip access_token auth on UpsertClaims route.
//		case path == "/v1/claims" && method == http.MethodPost:
//			return true
//		// Skip access_token auth on insertAnalysis route.
//		case regexp.MustCompile(`^/v1/claims/([a-zA-Z0-9\-_]+)/items/([a-zA-Z0-9\-_]+)/analysis_v0$`).MatchString(path) && method == http.MethodPost:
//			return true
//		default:
//			return false
//		}
//	})
//}
