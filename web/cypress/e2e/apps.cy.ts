/// <reference types="cypress" />

describe('App Catalog', () => {
  beforeEach(() => {
    // Login first
    cy.visit('/login');
    cy.get('input[name="username"]').type('admin@example.com');
    cy.get('input[name="password"]').type('admin123');
    cy.get('button[type="submit"]').click();
    cy.url().should('not.include', '/login');
    
    // Navigate to apps
    cy.visit('/apps');
  });

  it('should display the app catalog', () => {
    // Check page title
    cy.contains('h1', 'App Catalog').should('be.visible');
    
    // Check tabs
    cy.get('[role="tablist"]').within(() => {
      cy.contains('Catalog').should('be.visible');
      cy.contains('Installed').should('be.visible');
    });
    
    // Check for catalog entries
    cy.get('[data-testid="app-card"]').should('have.length.greaterThan', 0);
  });

  it('should filter apps by category', () => {
    // Click on a category chip
    cy.contains('button', 'testing').click();
    
    // Verify filtered results
    cy.get('[data-testid="app-card"]').each(($card) => {
      cy.wrap($card).should('contain', 'testing');
    });
  });

  it('should search for apps', () => {
    // Type in search box
    cy.get('input[placeholder*="Search"]').type('whoami');
    
    // Verify search results
    cy.get('[data-testid="app-card"]').should('have.length', 1);
    cy.get('[data-testid="app-card"]').should('contain', 'Whoami');
  });

  it('should navigate to install wizard', () => {
    // Click install on whoami app
    cy.get('[data-testid="app-card"]')
      .contains('Whoami')
      .parents('[data-testid="app-card"]')
      .find('button')
      .contains('Install')
      .click();
    
    // Verify we're on the install page
    cy.url().should('include', '/apps/install/whoami');
    cy.contains('h1', 'Install Whoami').should('be.visible');
  });
});

describe('App Installation', () => {
  beforeEach(() => {
    // Login and navigate to install page
    cy.visit('/login');
    cy.get('input[name="username"]').type('admin@example.com');
    cy.get('input[name="password"]').type('admin123');
    cy.get('button[type="submit"]').click();
    
    cy.visit('/apps/install/whoami');
  });

  it('should complete the installation wizard', () => {
    // Step 1: Overview
    cy.contains('About Whoami').should('be.visible');
    cy.contains('button', 'Next').click();
    
    // Step 2: Configuration
    cy.contains('Configuration').should('be.visible');
    cy.get('input[name="WHOAMI_PORT"]').clear().type('8090');
    cy.get('input[name="WHOAMI_NAME"]').clear().type('Test App');
    cy.contains('button', 'Next').click();
    
    // Step 3: Storage
    cy.contains('Storage Configuration').should('be.visible');
    cy.contains('/srv/apps/whoami/data').should('be.visible');
    cy.contains('button', 'Next').click();
    
    // Step 4: Review & Install
    cy.contains('Review & Install').should('be.visible');
    cy.contains('WHOAMI_PORT').should('be.visible');
    cy.contains('8090').should('be.visible');
    
    // Install the app
    cy.intercept('POST', '/api/v1/apps/install', {
      statusCode: 201,
      body: {
        message: 'App installed successfully',
        app: {
          id: 'whoami',
          name: 'Whoami',
          status: 'running'
        }
      }
    }).as('installApp');
    
    cy.contains('button', 'Install App').click();
    cy.wait('@installApp');
    
    // Should redirect to app details
    cy.url().should('include', '/apps/whoami');
  });

  it('should validate required fields', () => {
    // Navigate to Nextcloud install
    cy.visit('/apps/install/nextcloud');
    
    // Skip to configuration
    cy.contains('button', 'Next').click();
    
    // Try to proceed without filling required fields
    cy.contains('button', 'Next').click();
    
    // Should show validation errors
    cy.contains('This field is required').should('be.visible');
    
    // Fill required fields
    cy.get('input[name="NEXTCLOUD_ADMIN_PASSWORD"]').type('SecurePass123!');
    cy.get('input[name="POSTGRES_PASSWORD"]').type('DbPass123!');
    
    // Should be able to proceed now
    cy.contains('button', 'Next').click();
    cy.contains('Storage Configuration').should('be.visible');
  });
});

describe('App Management', () => {
  beforeEach(() => {
    // Login
    cy.visit('/login');
    cy.get('input[name="username"]').type('admin@example.com');
    cy.get('input[name="password"]').type('admin123');
    cy.get('button[type="submit"]').click();
    
    // Mock an installed app
    cy.intercept('GET', '/api/v1/apps/whoami', {
      statusCode: 200,
      body: {
        id: 'whoami',
        name: 'Whoami',
        version: 'latest',
        status: 'running',
        health: {
          status: 'healthy',
          checked_at: new Date().toISOString()
        },
        urls: ['http://localhost:8090'],
        ports: [{ host: 8090, container: 80, protocol: 'tcp' }],
        snapshots: []
      }
    }).as('getApp');
    
    cy.visit('/apps/whoami');
    cy.wait('@getApp');
  });

  it('should display app details', () => {
    // Check header
    cy.contains('h1', 'Whoami').should('be.visible');
    cy.contains('Running').should('be.visible');
    cy.contains('Healthy').should('be.visible');
    
    // Check action buttons
    cy.contains('button', 'Restart').should('be.visible');
    cy.contains('button', 'Stop').should('be.visible');
    cy.contains('a', 'Open App').should('have.attr', 'href', 'http://localhost:8090');
  });

  it('should stop and start the app', () => {
    // Stop the app
    cy.intercept('POST', '/api/v1/apps/whoami/stop', {
      statusCode: 200,
      body: { message: 'App stopped' }
    }).as('stopApp');
    
    cy.contains('button', 'Stop').click();
    cy.wait('@stopApp');
    
    // Update status to stopped
    cy.intercept('GET', '/api/v1/apps/whoami', {
      statusCode: 200,
      body: {
        id: 'whoami',
        name: 'Whoami',
        status: 'stopped',
        health: { status: 'unknown' }
      }
    }).as('getStoppedApp');
    
    cy.wait('@getStoppedApp');
    cy.contains('Stopped').should('be.visible');
    cy.contains('button', 'Start').should('be.visible');
    
    // Start the app
    cy.intercept('POST', '/api/v1/apps/whoami/start', {
      statusCode: 200,
      body: { message: 'App started' }
    }).as('startApp');
    
    cy.contains('button', 'Start').click();
    cy.wait('@startApp');
  });

  it('should show and navigate tabs', () => {
    // Check all tabs are present
    cy.contains('button', 'Health').should('be.visible');
    cy.contains('button', 'Logs').should('be.visible');
    cy.contains('button', 'Config').should('be.visible');
    cy.contains('button', 'Snapshots').should('be.visible');
    
    // Navigate to logs tab
    cy.contains('button', 'Logs').click();
    cy.contains('h2', 'Logs').should('be.visible');
    
    // Navigate to config tab
    cy.contains('button', 'Config').click();
    cy.contains('h2', 'Configuration').should('be.visible');
    
    // Navigate to snapshots tab
    cy.contains('button', 'Snapshots').click();
    cy.contains('h2', 'Snapshots').should('be.visible');
  });

  it('should perform health check', () => {
    cy.intercept('POST', '/api/v1/apps/whoami/health', {
      statusCode: 200,
      body: {
        message: 'Health check completed',
        health: {
          status: 'healthy',
          checked_at: new Date().toISOString()
        }
      }
    }).as('healthCheck');
    
    cy.contains('button', 'Check Now').click();
    cy.wait('@healthCheck');
    
    // Should show success message
    cy.contains('Health check completed').should('be.visible');
  });

  it('should delete the app', () => {
    // Open danger zone
    cy.contains('Delete App').click();
    
    // Check delete confirmation
    cy.contains('This will stop and remove the app').should('be.visible');
    cy.get('input[type="checkbox"]').check(); // Keep data
    
    // Confirm deletion
    cy.intercept('DELETE', '/api/v1/apps/whoami?keep_data=true', {
      statusCode: 200,
      body: { message: 'App deleted' }
    }).as('deleteApp');
    
    cy.contains('button', 'Confirm Delete').click();
    cy.wait('@deleteApp');
    
    // Should redirect to apps list
    cy.url().should('include', '/apps');
    cy.url().should('not.include', '/whoami');
  });
});

describe('Installed Apps View', () => {
  beforeEach(() => {
    // Login
    cy.visit('/login');
    cy.get('input[name="username"]').type('admin@example.com');
    cy.get('input[name="password"]').type('admin123');
    cy.get('button[type="submit"]').click();
    
    // Mock installed apps
    cy.intercept('GET', '/api/v1/apps/installed', {
      statusCode: 200,
      body: {
        items: [
          {
            id: 'whoami',
            name: 'Whoami',
            version: 'latest',
            status: 'running',
            health: { status: 'healthy' },
            urls: ['http://localhost:8090']
          },
          {
            id: 'nextcloud',
            name: 'Nextcloud',
            version: '28.0',
            status: 'stopped',
            health: { status: 'unknown' },
            urls: []
          }
        ]
      }
    }).as('getInstalled');
    
    cy.visit('/apps');
    cy.wait('@getInstalled');
  });

  it('should show installed apps', () => {
    // Switch to installed tab
    cy.contains('button', 'Installed').click();
    
    // Check installed apps are shown
    cy.contains('Whoami').should('be.visible');
    cy.contains('Running').should('be.visible');
    cy.contains('Healthy').should('be.visible');
    
    cy.contains('Nextcloud').should('be.visible');
    cy.contains('Stopped').should('be.visible');
  });

  it('should show quick actions for installed apps', () => {
    cy.contains('button', 'Installed').click();
    
    // Check actions for running app
    cy.contains('Whoami')
      .parents('[data-testid="installed-app-card"]')
      .within(() => {
        cy.get('button[aria-label="Restart"]').should('be.visible');
        cy.get('button[aria-label="Stop"]').should('be.visible');
        cy.get('a[aria-label="Open App"]').should('be.visible');
      });
    
    // Check actions for stopped app
    cy.contains('Nextcloud')
      .parents('[data-testid="installed-app-card"]')
      .within(() => {
        cy.get('button[aria-label="Start"]').should('be.visible');
      });
  });

  it('should handle health status correctly', () => {
    cy.contains('button', 'Installed').click();
    
    // Check health badges
    cy.contains('Whoami')
      .parents('[data-testid="installed-app-card"]')
      .should('contain', 'Healthy');
    
    // Simulate unhealthy app
    cy.intercept('GET', '/api/v1/apps/installed', {
      statusCode: 200,
      body: {
        items: [
          {
            id: 'whoami',
            name: 'Whoami',
            status: 'running',
            health: { status: 'unhealthy', message: 'Container restart loop' }
          }
        ]
      }
    }).as('getUnhealthy');
    
    // Refresh
    cy.reload();
    cy.wait('@getUnhealthy');
    cy.contains('button', 'Installed').click();
    
    // Should show unhealthy status
    cy.contains('Unhealthy').should('be.visible');
  });
});
