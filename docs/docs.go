// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/gitea": {
            "post": {
                "description": "Receive Gitea hook",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "WebHook"
                ],
                "summary": "Receive Gitea hook",
                "parameters": [
                    {
                        "description": "Gitea Hook",
                        "name": "hook",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handlers.WebhookPayload"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "allOf": [
                                {
                                    "$ref": "#/definitions/handlers.ResponseHTTP"
                                },
                                {
                                    "type": "object",
                                    "properties": {
                                        "type": {
                                            "$ref": "#/definitions/handlers.WebhookPayload"
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "503": {
                        "description": "Service Unavailable",
                        "schema": {
                            "$ref": "#/definitions/handlers.ResponseHTTP"
                        }
                    }
                }
            }
        },
        "/api/sandbox": {
            "post": {
                "description": "Specify the shell command for the corresponding repo",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Sandbox"
                ],
                "summary": "Specify the shell command for the corresponding repo",
                "parameters": [
                    {
                        "description": "Shell command",
                        "name": "cmd",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.Sandbox"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "allOf": [
                                {
                                    "$ref": "#/definitions/handlers.ResponseHTTP"
                                },
                                {
                                    "type": "object",
                                    "properties": {
                                        "data": {
                                            "$ref": "#/definitions/models.Sandbox"
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "503": {
                        "description": "Service Unavailable",
                        "schema": {
                            "$ref": "#/definitions/handlers.ResponseHTTP"
                        }
                    }
                }
            }
        },
        "/api/sandbox/status": {
            "get": {
                "description": "Get the current available sandbox count and waiting count",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Sandbox"
                ],
                "summary": "Get the current available sandbox count and waiting count",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handlers.StatusResponse"
                        }
                    },
                    "500": {
                        "description": "Sandbox instance not initialized",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/score": {
            "get": {
                "description": "Get a score by repo",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Score"
                ],
                "summary": "Get a score by repo",
                "parameters": [
                    {
                        "type": "string",
                        "description": "owner of the repo",
                        "name": "owner",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "name of the repo",
                        "name": "repo",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "page number of results to return (1-based)",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "page size of results. Default is 10.",
                        "name": "limit",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "allOf": [
                                {
                                    "$ref": "#/definitions/handlers.ResponseHTTP"
                                },
                                {
                                    "type": "object",
                                    "properties": {
                                        "data": {
                                            "$ref": "#/definitions/models.Score"
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "$ref": "#/definitions/handlers.ResponseHTTP"
                        }
                    },
                    "503": {
                        "description": "Service Unavailable",
                        "schema": {
                            "$ref": "#/definitions/handlers.ResponseHTTP"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "gitea.Commit": {
            "type": "object",
            "properties": {
                "author": {
                    "$ref": "#/definitions/gitea.User"
                },
                "commit": {
                    "$ref": "#/definitions/gitea.RepoCommit"
                },
                "committer": {
                    "$ref": "#/definitions/gitea.User"
                },
                "created": {
                    "type": "string"
                },
                "files": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/gitea.CommitAffectedFiles"
                    }
                },
                "html_url": {
                    "type": "string"
                },
                "parents": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/gitea.CommitMeta"
                    }
                },
                "sha": {
                    "type": "string"
                },
                "stats": {
                    "$ref": "#/definitions/gitea.CommitStats"
                },
                "url": {
                    "type": "string"
                }
            }
        },
        "gitea.CommitAffectedFiles": {
            "type": "object",
            "properties": {
                "filename": {
                    "type": "string"
                }
            }
        },
        "gitea.CommitMeta": {
            "type": "object",
            "properties": {
                "created": {
                    "type": "string"
                },
                "sha": {
                    "type": "string"
                },
                "url": {
                    "type": "string"
                }
            }
        },
        "gitea.CommitStats": {
            "type": "object",
            "properties": {
                "additions": {
                    "type": "integer"
                },
                "deletions": {
                    "type": "integer"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "gitea.CommitUser": {
            "type": "object",
            "properties": {
                "date": {
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                }
            }
        },
        "gitea.ExternalTracker": {
            "type": "object",
            "properties": {
                "external_tracker_format": {
                    "description": "External Issue Tracker URL Format. Use the placeholders {user}, {repo} and {index} for the username, repository name and issue index.",
                    "type": "string"
                },
                "external_tracker_style": {
                    "description": "External Issue Tracker Number Format, either ` + "`" + `numeric` + "`" + ` or ` + "`" + `alphanumeric` + "`" + `",
                    "type": "string"
                },
                "external_tracker_url": {
                    "description": "URL of external issue tracker.",
                    "type": "string"
                }
            }
        },
        "gitea.ExternalWiki": {
            "type": "object",
            "properties": {
                "external_wiki_url": {
                    "description": "URL of external wiki.",
                    "type": "string"
                }
            }
        },
        "gitea.InternalTracker": {
            "type": "object",
            "properties": {
                "allow_only_contributors_to_track_time": {
                    "description": "Let only contributors track time (Built-in issue tracker)",
                    "type": "boolean"
                },
                "enable_issue_dependencies": {
                    "description": "Enable dependencies for issues and pull requests (Built-in issue tracker)",
                    "type": "boolean"
                },
                "enable_time_tracker": {
                    "description": "Enable time tracking (Built-in issue tracker)",
                    "type": "boolean"
                }
            }
        },
        "gitea.MergeStyle": {
            "type": "string",
            "enum": [
                "merge",
                "rebase",
                "rebase-merge",
                "squash"
            ],
            "x-enum-varnames": [
                "MergeStyleMerge",
                "MergeStyleRebase",
                "MergeStyleRebaseMerge",
                "MergeStyleSquash"
            ]
        },
        "gitea.PayloadCommitVerification": {
            "type": "object",
            "properties": {
                "payload": {
                    "type": "string"
                },
                "reason": {
                    "type": "string"
                },
                "signature": {
                    "type": "string"
                },
                "verified": {
                    "type": "boolean"
                }
            }
        },
        "gitea.Permission": {
            "type": "object",
            "properties": {
                "admin": {
                    "type": "boolean"
                },
                "pull": {
                    "type": "boolean"
                },
                "push": {
                    "type": "boolean"
                }
            }
        },
        "gitea.RepoCommit": {
            "type": "object",
            "properties": {
                "author": {
                    "$ref": "#/definitions/gitea.CommitUser"
                },
                "committer": {
                    "$ref": "#/definitions/gitea.CommitUser"
                },
                "message": {
                    "type": "string"
                },
                "tree": {
                    "$ref": "#/definitions/gitea.CommitMeta"
                },
                "url": {
                    "type": "string"
                },
                "verification": {
                    "$ref": "#/definitions/gitea.PayloadCommitVerification"
                }
            }
        },
        "gitea.Repository": {
            "type": "object",
            "properties": {
                "allow_merge_commits": {
                    "type": "boolean"
                },
                "allow_rebase": {
                    "type": "boolean"
                },
                "allow_rebase_explicit": {
                    "type": "boolean"
                },
                "allow_squash_merge": {
                    "type": "boolean"
                },
                "archived": {
                    "type": "boolean"
                },
                "avatar_url": {
                    "type": "string"
                },
                "clone_url": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "default_branch": {
                    "type": "string"
                },
                "default_merge_style": {
                    "$ref": "#/definitions/gitea.MergeStyle"
                },
                "description": {
                    "type": "string"
                },
                "empty": {
                    "type": "boolean"
                },
                "external_tracker": {
                    "$ref": "#/definitions/gitea.ExternalTracker"
                },
                "external_wiki": {
                    "$ref": "#/definitions/gitea.ExternalWiki"
                },
                "fork": {
                    "type": "boolean"
                },
                "forks_count": {
                    "type": "integer"
                },
                "full_name": {
                    "type": "string"
                },
                "has_actions": {
                    "type": "boolean"
                },
                "has_issues": {
                    "type": "boolean"
                },
                "has_packages": {
                    "type": "boolean"
                },
                "has_projects": {
                    "type": "boolean"
                },
                "has_pull_requests": {
                    "type": "boolean"
                },
                "has_releases": {
                    "type": "boolean"
                },
                "has_wiki": {
                    "type": "boolean"
                },
                "html_url": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "ignore_whitespace_conflicts": {
                    "type": "boolean"
                },
                "internal": {
                    "type": "boolean"
                },
                "internal_tracker": {
                    "$ref": "#/definitions/gitea.InternalTracker"
                },
                "mirror": {
                    "type": "boolean"
                },
                "mirror_interval": {
                    "type": "string"
                },
                "mirror_updated": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "open_issues_count": {
                    "type": "integer"
                },
                "open_pr_counter": {
                    "type": "integer"
                },
                "original_url": {
                    "type": "string"
                },
                "owner": {
                    "$ref": "#/definitions/gitea.User"
                },
                "parent": {
                    "$ref": "#/definitions/gitea.Repository"
                },
                "permissions": {
                    "$ref": "#/definitions/gitea.Permission"
                },
                "private": {
                    "type": "boolean"
                },
                "release_counter": {
                    "type": "integer"
                },
                "size": {
                    "type": "integer"
                },
                "ssh_url": {
                    "type": "string"
                },
                "stars_count": {
                    "type": "integer"
                },
                "template": {
                    "type": "boolean"
                },
                "updated_at": {
                    "type": "string"
                },
                "watchers_count": {
                    "type": "integer"
                },
                "website": {
                    "type": "string"
                }
            }
        },
        "gitea.User": {
            "type": "object",
            "properties": {
                "active": {
                    "description": "Is user active",
                    "type": "boolean"
                },
                "avatar_url": {
                    "description": "URL to the user's avatar",
                    "type": "string"
                },
                "created": {
                    "description": "Date and Time of user creation",
                    "type": "string"
                },
                "description": {
                    "description": "the user's description",
                    "type": "string"
                },
                "email": {
                    "type": "string"
                },
                "followers_count": {
                    "description": "user counts",
                    "type": "integer"
                },
                "following_count": {
                    "type": "integer"
                },
                "full_name": {
                    "description": "the user's full name",
                    "type": "string"
                },
                "id": {
                    "description": "the user's id",
                    "type": "integer"
                },
                "is_admin": {
                    "description": "Is the user an administrator",
                    "type": "boolean"
                },
                "language": {
                    "description": "User locale",
                    "type": "string"
                },
                "last_login": {
                    "description": "Date and Time of last login",
                    "type": "string"
                },
                "location": {
                    "description": "the user's location",
                    "type": "string"
                },
                "login": {
                    "description": "the user's username",
                    "type": "string"
                },
                "login_name": {
                    "description": "The login_name of non local users (e.g. LDAP / OAuth / SMTP)",
                    "type": "string"
                },
                "prohibit_login": {
                    "description": "Is user login prohibited",
                    "type": "boolean"
                },
                "restricted": {
                    "description": "Is user restricted",
                    "type": "boolean"
                },
                "source_id": {
                    "description": "The ID of the Authentication Source for non local users.",
                    "type": "integer"
                },
                "starred_repos_count": {
                    "type": "integer"
                },
                "visibility": {
                    "description": "User visibility level option",
                    "allOf": [
                        {
                            "$ref": "#/definitions/gitea.VisibleType"
                        }
                    ]
                },
                "website": {
                    "description": "the user's website",
                    "type": "string"
                }
            }
        },
        "gitea.VisibleType": {
            "type": "string",
            "enum": [
                "public",
                "limited",
                "private"
            ],
            "x-enum-varnames": [
                "VisibleTypePublic",
                "VisibleTypeLimited",
                "VisibleTypePrivate"
            ]
        },
        "handlers.ResponseHTTP": {
            "type": "object",
            "properties": {
                "data": {},
                "message": {
                    "type": "string"
                },
                "success": {
                    "type": "boolean"
                }
            }
        },
        "handlers.StatusResponse": {
            "type": "object",
            "properties": {
                "available_count": {
                    "type": "integer"
                },
                "processing_count": {
                    "type": "integer"
                },
                "waiting_count": {
                    "type": "integer"
                }
            }
        },
        "handlers.WebhookPayload": {
            "type": "object",
            "properties": {
                "after": {
                    "type": "string"
                },
                "before": {
                    "type": "string"
                },
                "commits": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/gitea.Commit"
                    }
                },
                "compare_url": {
                    "type": "string"
                },
                "pusher": {
                    "$ref": "#/definitions/gitea.User"
                },
                "ref": {
                    "type": "string"
                },
                "repository": {
                    "$ref": "#/definitions/gitea.Repository"
                },
                "sender": {
                    "$ref": "#/definitions/gitea.User"
                }
            }
        },
        "models.Sandbox": {
            "type": "object",
            "required": [
                "script",
                "source_git_url"
            ],
            "properties": {
                "script": {
                    "type": "string",
                    "example": "#!/bin/bash\n\necho 'Hello, World!'"
                },
                "source_git_url": {
                    "type": "string",
                    "example": "user_name/repo_name"
                }
            }
        },
        "models.Score": {
            "type": "object",
            "required": [
                "judge_time",
                "message",
                "score"
            ],
            "properties": {
                "judge_time": {
                    "type": "string",
                    "example": "2021-07-01T00:00:00Z"
                },
                "message": {
                    "type": "string",
                    "example": "Scored successfully"
                },
                "score": {
                    "type": "number",
                    "example": 100
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "OJ-PoC API",
	Description:      "This is a simple OJ-PoC API server.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
