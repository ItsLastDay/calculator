// swagger-ui-dist ships no type definitions; this covers the part we use.
declare module 'swagger-ui-dist/swagger-ui-es-bundle.js' {
  interface SwaggerUIOptions {
    url: string;
    dom_id: string;
    deepLinking?: boolean;
    tryItOutEnabled?: boolean;
  }

  const SwaggerUIBundle: (options: SwaggerUIOptions) => unknown;
  export default SwaggerUIBundle;
}
