package docs

import _ "embed"

//go:embed openapi.yaml
var Spec []byte

// SwaggerHTML — страница Swagger UI (скрипты подгружаются браузером с CDN).
const SwaggerHTML = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>ShopAI API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({ url: '/openapi.yaml', dom_id: '#swagger-ui', persistAuthorization: true });
  </script>
</body>
</html>`
