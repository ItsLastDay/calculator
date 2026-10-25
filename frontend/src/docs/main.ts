// Entry point of the /docs/ page: renders the contract the server publishes,
// so the documentation can never drift from the running API.
import SwaggerUIBundle from 'swagger-ui-dist/swagger-ui-es-bundle.js';
import 'swagger-ui-dist/swagger-ui.css';
import './docs.css';

SwaggerUIBundle({
  url: '/api/v1/openapi.yaml',
  dom_id: '#swagger-ui',
  deepLinking: true,
  tryItOutEnabled: true,
});
