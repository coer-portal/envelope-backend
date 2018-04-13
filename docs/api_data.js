define({ "api": [
  {
    "type": "post",
    "url": "/register-device",
    "title": "Register a Device",
    "name": "RegisterDevice",
    "group": "Device",
    "header": {
      "fields": {
        "Header": [
          {
            "group": "Header",
            "type": "String",
            "optional": false,
            "field": "deviceid",
            "description": "<p>Unique Device ID</p>"
          }
        ]
      }
    },
    "success": {
      "examples": [
        {
          "title": "Success-Example:",
          "content": "HTTP/1.1 200 Ok\n{\"hash\":\"XVlBzgbaiCMRAjWwhTHc\",\"status\":\"OK\",\"status_code\":200}",
          "type": "json"
        }
      ]
    },
    "error": {
      "examples": [
        {
          "title": "Error-Example:",
          "content": "HTTP/1.1 400 Bad Request\n{\"error_code\":\"NOT_FOUND\",\"status\":\"Bad Request\",\"status_code\":400}\n\nHTTP/1.1 500 Internal Server Error\n{\"error_code\":\"INTERNAL_ERROR\",\"status\":\"Internal Server Error\",\"status_code:\"500\"}\n\nHTTP/1.1 401 Unauthorized\n{\"error_code\":\"OUT_OF_REGION\",\"status\":\"Unauthorized\",\"status_code:\"401\"}",
          "type": "json"
        }
      ]
    },
    "version": "0.0.0",
    "filename": "./router/router.go",
    "groupTitle": "Device"
  },
  {
    "type": "get",
    "url": "/verify-device",
    "title": "Verify a Device",
    "name": "VerifyDevice",
    "group": "Device",
    "header": {
      "fields": {
        "Header": [
          {
            "group": "Header",
            "type": "String",
            "optional": false,
            "field": "deviceid",
            "description": "<p>Unique Device ID</p>"
          }
        ]
      }
    },
    "success": {
      "examples": [
        {
          "title": "Success-Example:",
          "content": "HTTP/1.1 200 Ok\n{\"status\":\"OK\",\"status_code\":200}",
          "type": "json"
        }
      ]
    },
    "error": {
      "examples": [
        {
          "title": "Error-Example:",
          "content": "HTTP/1.1 400 Bad Request\n{\"error_code\":\"EXPIRED\",\"status\":\"Bad Request\",\"status_code\":400}\n\nHTTP/1.1 500 Internal Server Error\n{\"error_code\":\"INTERNAL_ERROR\",\"status\":\"Internal Server Error\",\"status_code:\"500\"}\n\nHTTP/1.1 401 Unauthorized\n{\"error_code\":\"NOT_REGISTERED\",\"status\":\"Unauthorized\",\"status_code:\"401\"}",
          "type": "json"
        }
      ]
    },
    "version": "0.0.0",
    "filename": "./router/router.go",
    "groupTitle": "Device"
  }
] });
