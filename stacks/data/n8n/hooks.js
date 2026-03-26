// Universe 647 — n8n Authelia Trusted Header SSO
// Reads Remote-Email header set by Caddy forward_auth → Authelia
// Auto-authenticates the user without showing n8n's login page.
//
// SECURITY: n8n must ONLY be reachable via Caddy (no host ports).
// Anyone who can reach n8n directly could spoof the header.

const HEADER = 'remote-email';

// Paths that should never be intercepted by SSO
const SKIP_PREFIXES = [
  '/healthz',
  '/webhook',
  '/webhook-test',
  '/webhook-waiting',
  '/rest/oauth2-credential',
  '/assets/',
  '/favicon',
];

module.exports = {
  'n8n.ready': [
    async function (hookData, app) {
      const { Container } = require('@n8n/di');

      // Import n8n internals for user lookup + cookie issuance
      const { UserRepository } = require(
        '@n8n/db/repositories/user.repository'
      );
      const { AuthService } = require('@n8n/api/services/auth.service');

      const userRepo = Container.get(UserRepository);
      const authService = Container.get(AuthService);

      // Insert middleware before all other handlers
      app.use(async (req, res, next) => {
        // Skip paths that don't need auth (webhooks, assets, health)
        const path = req.path.toLowerCase();
        if (SKIP_PREFIXES.some((p) => path.startsWith(p))) {
          return next();
        }

        // Skip if user already has a valid session cookie
        if (req.cookies?.['n8n-auth']) {
          return next();
        }

        const email = req.headers[HEADER];
        if (!email) {
          return next();
        }

        try {
          const user = await userRepo.findOne({
            where: { email: email.toLowerCase() },
          });

          if (user) {
            authService.issueCookie(res, user, req.browserId);
          }
        } catch (err) {
          console.error('[authelia-sso] Failed to auto-login:', err.message);
        }

        return next();
      });

      console.log('[authelia-sso] Trusted header SSO hook loaded');
    },
  ],
};
