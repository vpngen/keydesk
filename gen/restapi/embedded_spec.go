// Code generated by go-swagger; DO NOT EDIT.

package restapi

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
)

var (
	// SwaggerJSON embedded version of the swagger document used at generation time
	SwaggerJSON json.RawMessage
	// FlatSwaggerJSON embedded flattened version of the swagger document used at generation time
	FlatSwaggerJSON json.RawMessage
)

func init() {
	SwaggerJSON = json.RawMessage([]byte(`{
  "swagger": "2.0",
  "info": {
    "title": "Keydesk server",
    "version": "1.0.0"
  },
  "basePath": "/",
  "paths": {
    "/token": {
      "post": {
        "produces": [
          "application/json"
        ],
        "responses": {
          "201": {
            "description": "Token created.",
            "schema": {
              "$ref": "#/definitions/token"
            }
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      }
    },
    "/user": {
      "get": {
        "security": [
          {
            "Bearer": []
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "A list of users.",
            "schema": {
              "type": "array",
              "items": {
                "$ref": "#/definitions/user"
              }
            }
          },
          "403": {
            "description": "You do not have necessary permissions for the resource"
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      },
      "post": {
        "security": [
          {
            "Bearer": []
          }
        ],
        "produces": [
          "application/octet-stream"
        ],
        "responses": {
          "201": {
            "description": "New user created.",
            "schema": {
              "type": "file"
            },
            "headers": {
              "Content-Disposition": {
                "type": "string",
                "description": "the value is ` + "`" + `attachment; filename=\"wg.conf\"` + "`" + `"
              }
            }
          },
          "403": {
            "description": "You do not have necessary permissions for the resource"
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      }
    },
    "/user/{UserID}": {
      "delete": {
        "security": [
          {
            "Bearer": []
          }
        ],
        "produces": [
          "application/json"
        ],
        "parameters": [
          {
            "type": "string",
            "name": "UserID",
            "in": "path",
            "required": true
          }
        ],
        "responses": {
          "204": {
            "description": "User deleted."
          },
          "403": {
            "description": "You do not have necessary permissions for the resource"
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "error": {
      "type": "object",
      "required": [
        "message"
      ],
      "properties": {
        "code": {
          "type": "integer"
        },
        "message": {
          "type": "string"
        }
      }
    },
    "token": {
      "type": "object",
      "required": [
        "Token"
      ],
      "properties": {
        "Token": {
          "type": "string"
        }
      }
    },
    "user": {
      "type": "object",
      "required": [
        "UserID",
        "UserName"
      ],
      "properties": {
        "LastVisitASCountry": {
          "type": "string"
        },
        "LastVisitASName": {
          "type": "string"
        },
        "LastVisitHour": {
          "type": "string",
          "format": "date-time",
          "x-nullable": true
        },
        "LastVisitSubnet": {
          "type": "string"
        },
        "MonthlyQuotaRemainingGB": {
          "type": "number",
          "format": "float"
        },
        "PersonDesc": {
          "type": "string"
        },
        "PersonDescLink": {
          "type": "string"
        },
        "PersonName": {
          "type": "string"
        },
        "Problems": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "Status": {
          "type": "string"
        },
        "ThrottlingTill": {
          "type": "string",
          "format": "date-time",
          "x-nullable": true
        },
        "UserID": {
          "type": "string"
        },
        "UserName": {
          "type": "string"
        }
      }
    }
  },
  "securityDefinitions": {
    "Bearer": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header"
    }
  }
}`))
	FlatSwaggerJSON = json.RawMessage([]byte(`{
  "swagger": "2.0",
  "info": {
    "title": "Keydesk server",
    "version": "1.0.0"
  },
  "basePath": "/",
  "paths": {
    "/token": {
      "post": {
        "produces": [
          "application/json"
        ],
        "responses": {
          "201": {
            "description": "Token created.",
            "schema": {
              "$ref": "#/definitions/token"
            }
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      }
    },
    "/user": {
      "get": {
        "security": [
          {
            "Bearer": []
          }
        ],
        "produces": [
          "application/json"
        ],
        "responses": {
          "200": {
            "description": "A list of users.",
            "schema": {
              "type": "array",
              "items": {
                "$ref": "#/definitions/user"
              }
            }
          },
          "403": {
            "description": "You do not have necessary permissions for the resource"
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      },
      "post": {
        "security": [
          {
            "Bearer": []
          }
        ],
        "produces": [
          "application/octet-stream"
        ],
        "responses": {
          "201": {
            "description": "New user created.",
            "schema": {
              "type": "file"
            },
            "headers": {
              "Content-Disposition": {
                "type": "string",
                "description": "the value is ` + "`" + `attachment; filename=\"wg.conf\"` + "`" + `"
              }
            }
          },
          "403": {
            "description": "You do not have necessary permissions for the resource"
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      }
    },
    "/user/{UserID}": {
      "delete": {
        "security": [
          {
            "Bearer": []
          }
        ],
        "produces": [
          "application/json"
        ],
        "parameters": [
          {
            "type": "string",
            "name": "UserID",
            "in": "path",
            "required": true
          }
        ],
        "responses": {
          "204": {
            "description": "User deleted."
          },
          "403": {
            "description": "You do not have necessary permissions for the resource"
          },
          "default": {
            "description": "error",
            "schema": {
              "$ref": "#/definitions/error"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "error": {
      "type": "object",
      "required": [
        "message"
      ],
      "properties": {
        "code": {
          "type": "integer"
        },
        "message": {
          "type": "string"
        }
      }
    },
    "token": {
      "type": "object",
      "required": [
        "Token"
      ],
      "properties": {
        "Token": {
          "type": "string"
        }
      }
    },
    "user": {
      "type": "object",
      "required": [
        "UserID",
        "UserName"
      ],
      "properties": {
        "LastVisitASCountry": {
          "type": "string"
        },
        "LastVisitASName": {
          "type": "string"
        },
        "LastVisitHour": {
          "type": "string",
          "format": "date-time",
          "x-nullable": true
        },
        "LastVisitSubnet": {
          "type": "string"
        },
        "MonthlyQuotaRemainingGB": {
          "type": "number",
          "format": "float"
        },
        "PersonDesc": {
          "type": "string"
        },
        "PersonDescLink": {
          "type": "string"
        },
        "PersonName": {
          "type": "string"
        },
        "Problems": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "Status": {
          "type": "string"
        },
        "ThrottlingTill": {
          "type": "string",
          "format": "date-time",
          "x-nullable": true
        },
        "UserID": {
          "type": "string"
        },
        "UserName": {
          "type": "string"
        }
      }
    }
  },
  "securityDefinitions": {
    "Bearer": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header"
    }
  }
}`))
}
