package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/eugen/termviewer/server/backend/pkg/config"
)

type KeycloakClient struct {
	Client *gocloak.GoCloak
	Config *config.Config
	Token  *gocloak.JWT
}

func InitKeycloak(cfg *config.Config) *KeycloakClient {
	log.Printf("Initializing Keycloak client at %s...", cfg.KeycloakURL)
	client := gocloak.NewClient(cfg.KeycloakURL)
	if cfg.KeycloakSkipTLSVerify {
		client.RestyClient().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	kc := &KeycloakClient{
		Client: client,
		Config: cfg,
	}

	// Retry logic for Keycloak startup
	var err error
	for i := 0; i < 10; i++ {
		err = kc.LoginAdmin()
		if err == nil {
			break
		}
		log.Printf("Waiting for Keycloak to be ready... (%d/10)", i+1)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to login to Keycloak admin after retries: %v", err)
	}

	// Ensure our realm exists
	err = kc.EnsureRealm()
	if err != nil {
		log.Fatalf("Failed to ensure realm: %v", err)
	}

	err = kc.EnsureClient()
	if err != nil {
		log.Fatalf("Failed to ensure client: %v", err)
	}

	err = kc.EnsureRealmRole(cfg.KeycloakAdminRole)
	if err != nil {
		log.Fatalf("Failed to ensure admin role: %v", err)
	}

	// Automatically create the initial admin user in the realm
	log.Printf("Checking initial admin user in realm...")
	err = kc.EnsureAdminUser()
	if err != nil {
		log.Printf("Error: Failed to ensure initial admin user: %v", err)
	} else {
		log.Printf("Initial admin user check complete.")
	}

	return kc
}

func (kc *KeycloakClient) EnsureAdminUser() error {
	if err := kc.EnsureAdminToken(); err != nil {
		return err
	}
	ctx := context.Background()

	// Check if the admin user already exists in the termviewer realm (Exact match)
	users, err := kc.Client.GetUsers(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, gocloak.GetUsersParams{
		Username: gocloak.StringP(kc.Config.KeycloakAdminUser),
		Exact:    gocloak.BoolP(true),
	})
	if err != nil {
		return err
	}

	var userID string
	if len(users) == 0 {
		log.Printf("Initial admin user not found, creating...")
		user := gocloak.User{
			Username:        gocloak.StringP(kc.Config.KeycloakAdminUser),
			Email:           gocloak.StringP(kc.Config.KeycloakAdminUser + "@termviewer.local"),
			FirstName:       gocloak.StringP("TermViewer"),
			LastName:        gocloak.StringP("Admin"),
			Enabled:         gocloak.BoolP(true),
			EmailVerified:   gocloak.BoolP(true),
			RequiredActions: &[]string{}, 
		}
		userID, err = kc.Client.CreateUser(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, user)
		if err != nil {
			return fmt.Errorf("failed to create initial admin user: %v", err)
		}

		err = kc.Client.SetPassword(ctx, kc.Token.AccessToken, userID, kc.Config.KeycloakRealm, kc.Config.KeycloakAdminPass, false)
		if err != nil {
			return fmt.Errorf("failed to set initial admin password: %v", err)
		}
		log.Printf("Initial admin user created successfully.")
	} else {
		userID = *users[0].ID
		log.Printf("Found existing admin user. Forcing status reset...")
		
		// Force update existing user to be fully functional
		user := gocloak.User{
			ID:              gocloak.StringP(userID),
			FirstName:       gocloak.StringP("TermViewer"),
			LastName:        gocloak.StringP("Admin"),
			Enabled:         gocloak.BoolP(true),
			EmailVerified:   gocloak.BoolP(true),
			RequiredActions: &[]string{}, // CRITICAL: This clears "Update Password", "Verify Email", etc.
		}
		err = kc.Client.UpdateUser(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, user)
		if err != nil {
			log.Printf("Warning: Failed to force-enable existing admin: %v", err)
		}

		// Re-sync password and ensure it is NOT temporary
		err = kc.Client.SetPassword(ctx, kc.Token.AccessToken, userID, kc.Config.KeycloakRealm, kc.Config.KeycloakAdminPass, false)
		if err != nil {
			log.Printf("Warning: Failed to sync admin password: %v", err)
		}
		log.Printf("Admin user status and password synchronized.")
	}

	// Ensure the user has the admin role
	hasRole, err := kc.UserHasRealmRole(userID, kc.Config.KeycloakAdminRole)
	if err != nil {
		return err
	}

	if !hasRole {
		fmt.Printf("Assigning role %q to initial admin user %q\n", kc.Config.KeycloakAdminRole, kc.Config.KeycloakAdminUser)
		role, err := kc.Client.GetRealmRole(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, kc.Config.KeycloakAdminRole)
		if err != nil {
			return err
		}
		err = kc.Client.AddRealmRoleToUser(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, userID, []gocloak.Role{*role})
		if err != nil {
			return fmt.Errorf("failed to assign admin role: %v", err)
		}
	}

	return nil
}

func (kc *KeycloakClient) LoginAdmin() error {
	ctx := context.Background()
	token, err := kc.Client.LoginAdmin(ctx, kc.Config.KeycloakAdminUser, kc.Config.KeycloakAdminPass, "master")
	if err != nil {
		return err
	}
	kc.Token = token
	return nil
}

func (kc *KeycloakClient) EnsureAdminToken() error {
	ctx := context.Background()
	if kc.Token == nil {
		return kc.LoginAdmin()
	}
	// Check token by calling a simple method
	_, err := kc.Client.GetRealm(ctx, kc.Token.AccessToken, "master")
	if err != nil {
		return kc.LoginAdmin()
	}
	return nil
}

func (kc *KeycloakClient) EnsureClient() error {
	if err := kc.EnsureAdminToken(); err != nil {
		return err
	}
	ctx := context.Background()
	clients, err := kc.Client.GetClients(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, gocloak.GetClientsParams{
		ClientID: gocloak.StringP(kc.Config.KeycloakAppClientID),
	})
	if err != nil {
		return err
	}

	client := gocloak.Client{
		ClientID:                  gocloak.StringP(kc.Config.KeycloakAppClientID),
		Enabled:                   gocloak.BoolP(true),
		DirectAccessGrantsEnabled: gocloak.BoolP(true),
		PublicClient:              gocloak.BoolP(true),
		Protocol:                  gocloak.StringP("openid-connect"),
		RedirectURIs:              &[]string{"*"},
		WebOrigins:                &[]string{"*"},
	}

	if len(clients) == 0 {
		fmt.Printf("Client '%s' not found, creating...\n", kc.Config.KeycloakAppClientID)
		_, err = kc.Client.CreateClient(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, client)
		if err != nil {
			return fmt.Errorf("failed to create client: %v", err)
		}
	} else {
		client.ID = clients[0].ID
		err = kc.Client.UpdateClient(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, client)
		if err != nil {
			return fmt.Errorf("failed to update client: %v", err)
		}
	}
	return nil
}

func (kc *KeycloakClient) EnsureRealmRole(roleName string) error {
	if err := kc.EnsureAdminToken(); err != nil {
		return err
	}

	ctx := context.Background()
	_, err := kc.Client.GetRealmRole(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, roleName)
	if err == nil {
		return nil
	}

	_, err = kc.Client.CreateRealmRole(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, gocloak.Role{
		Name:        gocloak.StringP(roleName),
		Description: gocloak.StringP("TermViewer admin users can approve pending accounts and manage onboarding."),
	})
	if err != nil {
		return fmt.Errorf("failed to create realm role %q: %v", roleName, err)
	}

	return nil
}

func (kc *KeycloakClient) EnsureRealm() error {
	if err := kc.EnsureAdminToken(); err != nil {
		return err
	}
	ctx := context.Background()
	_, err := kc.Client.GetRealm(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm)
	
	realmConf := gocloak.RealmRepresentation{
		Realm:               gocloak.StringP(kc.Config.KeycloakRealm),
		Enabled:             gocloak.BoolP(true),
		RegistrationAllowed: gocloak.BoolP(true),
		VerifyEmail:         gocloak.BoolP(false),
		LoginWithEmailAllowed: gocloak.BoolP(true),
		ResetPasswordAllowed:  gocloak.BoolP(true),

		BruteForceProtected:   gocloak.BoolP(true),
		MaxFailureWaitSeconds: gocloak.IntP(900),
		MinimumQuickLoginWaitSeconds: gocloak.IntP(5),
		MaxDeltaTimeSeconds:   gocloak.IntP(60 * 60),
		FailureFactor:         gocloak.IntP(5),
	}
	if err != nil {
		log.Printf("Realm %s not found, creating...\n", kc.Config.KeycloakRealm)
		_, err = kc.Client.CreateRealm(ctx, kc.Token.AccessToken, realmConf)
		if err != nil {
			return fmt.Errorf("failed to create realm: %v", err)
		}
	} else {
		// Update existing realm to ensure settings are permissive
		err = kc.Client.UpdateRealm(ctx, kc.Token.AccessToken, realmConf)
		if err != nil {
			return fmt.Errorf("failed to update realm: %v", err)
		}
	}
	return nil
}

func (kc *KeycloakClient) CreateUser(username, email, password string) (string, error) {
	if err := kc.EnsureAdminToken(); err != nil {
		return "", err
	}
	ctx := context.Background()
	user := gocloak.User{
		Username:  gocloak.StringP(username),
		Email:     gocloak.StringP(email),
		FirstName: gocloak.StringP(username),
		LastName:  gocloak.StringP("User"),
		Enabled:   gocloak.BoolP(false), // Disabled by default until approved
	}

	userID, err := kc.Client.CreateUser(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, user)
	if err != nil {
		return "", err
	}

	err = kc.Client.SetPassword(ctx, kc.Token.AccessToken, userID, kc.Config.KeycloakRealm, password, false)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func (kc *KeycloakClient) EnableUser(userID string) error {
	if err := kc.EnsureAdminToken(); err != nil {
		return err
	}
	ctx := context.Background()
	user := gocloak.User{
		ID:              gocloak.StringP(userID),
		Enabled:         gocloak.BoolP(true),
		EmailVerified:   gocloak.BoolP(true),
		RequiredActions: &[]string{},
	}
	return kc.Client.UpdateUser(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, user)
}

func (kc *KeycloakClient) DeleteUser(userID string) error {
	if err := kc.EnsureAdminToken(); err != nil {
		return err
	}
	ctx := context.Background()
	return kc.Client.DeleteUser(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, userID)
}

func (kc *KeycloakClient) UserHasRealmRole(userID, roleName string) (bool, error) {
	if err := kc.EnsureAdminToken(); err != nil {
		return false, err
	}

	ctx := context.Background()
	roles, err := kc.Client.GetRealmRolesByUserID(ctx, kc.Token.AccessToken, kc.Config.KeycloakRealm, userID)
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		if role == nil || role.Name == nil {
			continue
		}
		if *role.Name == roleName {
			return true, nil
		}
	}

	return false, nil
}
