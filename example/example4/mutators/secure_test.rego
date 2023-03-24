package secure_test

import data.secure.patch


import future.keywords

test_inject if {


    ops := patch with input as {
    "Affinities": null,
    "AllAtOnce": false,
    "Constraints": null,
    "ConsulNamespace": "",
    "ConsulToken": "",
    "CreateIndex": 555,
    "Datacenters": [
      "dc1"
    ],
    "DispatchIdempotencyToken": "",
    "Dispatched": false,
    "ID": "webapp",
    "JobModifyIndex": 1609,
    "Meta": null,
    "ModifyIndex": 1609,
    "Multiregion": null,
    "Name": "webapp",
    "Namespace": "default",
    "NomadTokenID": "",
    "ParameterizedJob": null,
    "ParentID": "",
    "Payload": null,
    "Periodic": null,
    "Priority": 50,
    "Region": "global",
    "Spreads": null,
    "Stable": false,
    "Status": "running",
    "StatusDescription": "",
    "Stop": false,
    "SubmitTime": 1679690773841206000,
    "TaskGroups": [
      {
        "Affinities": null,
        "Constraints": [
          {
            "LTarget": "${attr.nomad.service_discovery}",
            "Operand": "=",
            "RTarget": "true"
          }
        ],
        "Consul": {
          "Namespace": ""
        },
        "Count": 1,
        "EphemeralDisk": {
          "Migrate": false,
          "SizeMB": 300,
          "Sticky": false
        },
        "MaxClientDisconnect": null,
        "Meta": {
          "secure": "webapp"
        },
        "Migrate": {
          "HealthCheck": "checks",
          "HealthyDeadline": 300000000000,
          "MaxParallel": 1,
          "MinHealthyTime": 10000000000
        },
        "Name": "webapp",
        "Networks": [
          {
            "CIDR": "",
            "DNS": null,
            "Device": "",
            "DynamicPorts": [
              {
                "HostNetwork": "default",
                "Label": "http",
                "To": 8000,
                "Value": 0
              }
            ],
            "IP": "",
            "MBits": 0,
            "Mode": "",
            "ReservedPorts": null
          }
        ],
        "ReschedulePolicy": {
          "Attempts": 0,
          "Delay": 30000000000,
          "DelayFunction": "exponential",
          "Interval": 0,
          "MaxDelay": 3600000000000,
          "Unlimited": true
        },
        "RestartPolicy": {
          "Attempts": 2,
          "Delay": 15000000000,
          "Interval": 1800000000000,
          "Mode": "fail"
        },
        "Scaling": null,
        "Services": [
          {
            "Address": "",
            "AddressMode": "auto",
            "CanaryMeta": null,
            "CanaryTags": null,
            "Checks": null,
            "Connect": null,
            "EnableTagOverride": false,
            "Meta": null,
            "Name": "webapp",
            "Namespace": "default",
            "OnUpdate": "require_healthy",
            "PortLabel": "http",
            "Provider": "nomad",
            "TaggedAddresses": null,
            "Tags": null,
            "TaskName": ""
          }
        ],
        "ShutdownDelay": null,
        "Spreads": null,
        "StopAfterClientDisconnect": null,
        "Tasks": [
          {
            "Affinities": null,
            "Artifacts": null,
            "CSIPluginConfig": null,
            "Config": {
              "args": [
                "/local/webapp.js"
              ],
              "image": "node:18-alpine",
              "ports": [
                "http"
              ]
            },
            "Constraints": null,
            "DispatchPayload": null,
            "Driver": "docker",
            "Env": null,
            "Identity": null,
            "KillSignal": "",
            "KillTimeout": 5000000000,
            "Kind": "",
            "Leader": false,
            "Lifecycle": null,
            "LogConfig": {
              "MaxFileSizeMB": 10,
              "MaxFiles": 10
            },
            "Meta": null,
            "Name": "webapp",
            "Resources": {
              "CPU": 100,
              "Cores": 0,
              "Devices": null,
              "DiskMB": 0,
              "IOPS": 0,
              "MemoryMB": 256,
              "MemoryMaxMB": 0,
              "Networks": null
            },
            "RestartPolicy": {
              "Attempts": 2,
              "Delay": 15000000000,
              "Interval": 1800000000000,
              "Mode": "fail"
            },
            "ScalingPolicies": null,
            "Services": null,
            "ShutdownDelay": 0,
            "Templates": [
              {
                "ChangeMode": "restart",
                "ChangeScript": null,
                "ChangeSignal": "",
                "DestPath": "local/webapp.js",
                "EmbeddedTmpl": "const http = require('node:http');\n\n// Create a local server to receive data from\nconst server = http.createServer((req, res) => {\n\n    console.log(\"Incoming request\", req.headers)\n\n    const username = req.headers[\"x-forwarded-preferred-username\"]\n    \n    res.writeHead(200, { 'Content-Type': 'application/json' });\n\n    res.end(JSON.stringify({\n        message: `Hello ${username || \"Unkown Somebody\"}!`\n    }));\n});\n\nserver.listen(8000);",
                "Envvars": false,
                "ErrMissingKey": false,
                "Gid": null,
                "LeftDelim": "{{",
                "Perms": "0644",
                "RightDelim": "}}",
                "SourcePath": "",
                "Splay": 5000000000,
                "Uid": null,
                "VaultGrace": 0,
                "Wait": null
              }
            ],
            "User": "",
            "Vault": null,
            "VolumeMounts": null
          }
        ],
        "Update": {
          "AutoPromote": false,
          "AutoRevert": false,
          "Canary": 0,
          "HealthCheck": "checks",
          "HealthyDeadline": 300000000000,
          "MaxParallel": 1,
          "MinHealthyTime": 10000000000,
          "ProgressDeadline": 600000000000,
          "Stagger": 30000000000
        },
        "Volumes": null
      }
    ],
    "Type": "service",
    "Update": {
      "AutoPromote": false,
      "AutoRevert": false,
      "Canary": 0,
      "HealthCheck": "",
      "HealthyDeadline": 0,
      "MaxParallel": 1,
      "MinHealthyTime": 0,
      "ProgressDeadline": 0,
      "Stagger": 30000000000
    },
    "VaultNamespace": "",
    "VaultToken": "",
    "Version": 29
  }


    print(ops)

    ops == {

  {
        "op": "add",
        "path": "/TaskGroups/0/Tasks/1",
        "value": {
            "Affinities": null,
            "Artifacts": null,
            "CSIPluginConfig": null,
            "Config": {
                "args": [
                    "--config",
                    "/local/config.cfg"
                ],
                "image": "bitnami/oauth2-proxy:7.4.0",
                "ports": [
                    "auth"
                ]
            },
            "Constraints": null,
            "DispatchPayload": null,
            "Driver": "docker",
            "Env": null,
            "Identity": null,
            "KillSignal": "",
            "KillTimeout": 5000000000,
            "Kind": "",
            "Leader": false,
            "Lifecycle": null,
            "LogConfig": {
                "MaxFileSizeMB": 10,
                "MaxFiles": 10
            },
            "Meta": null,
            "Name": "auth",
            "Resources": {
                "CPU": 100,
                "Cores": 0,
                "Devices": null,
                "DiskMB": 0,
                "IOPS": 0,
                "MemoryMB": 300,
                "MemoryMaxMB": 0,
                "Networks": null
            },
            "RestartPolicy": {
                "Attempts": 2,
                "Delay": 15000000000,
                "Interval": 1800000000000,
                "Mode": "fail"
            },
            "ScalingPolicies": null,
            "Services": null,
            "ShutdownDelay": 0,
            "Templates": [
                {
                    "ChangeMode": "restart",
                    "ChangeScript": null,
                    "ChangeSignal": "",
                    "DestPath": "${NOMAD_TASK_DIR}/config.cfg",
                    "EmbeddedTmpl": "provider = \"keycloak-oidc\"\nclient_id = \"webapp\"\nclient_secret = \"secret\"\nredirect_url = \"http://webapp.nomad.local/oauth2/callback\"\noidc_issuer_url = \"http://keycloak.nomad.local/realms/demo\"\nemail_domains = [\"*\"]\ncookie_secret=\"TEktZxl3wbO9cL3mkm-DyMRvjhhJqxf7Xk8fcZQFq-U=\"\nhttp_address=\"http://0.0.0.0:4180\"\noidc_extra_audiences=[\"account\"]\ncookie_secure=false\n{{ range nomadService \"internal-webapp\" }}\nupstreams = [\n  \"http://{{ .Address }}:{{ .Port }}\"\n]\n{{ end }}",
                    "Envvars": false,
                    "ErrMissingKey": false,
                    "Gid": null,
                    "LeftDelim": "{{",
                    "Perms": "0644",
                    "RightDelim": "}}",
                    "SourcePath": "",
                    "Splay": 5000000000,
                    "Uid": null,
                    "VaultGrace": 0,
                    "Wait": null
                }
            ],
            "User": "",
            "Vault": null,
            "VolumeMounts": null
        }
    },
    {
        "op": "replace",
        "path": "/TaskGroups/0/Services/0/Name",
        "value": "internal-webapp"
    },
    {
        "op": "add",
        "path": "/TaskGroups/0/Services/1",
        "value": {
            "Address": "",
            "AddressMode": "auto",
            "CanaryMeta": null,
            "CanaryTags": null,
            "Checks": null,
            "Connect": null,
            "EnableTagOverride": false,
            "Meta": null,
            "Name": "webapp",
            "Namespace": "default",
            "OnUpdate": "require_healthy",
            "PortLabel": "auth",
            "Provider": "nomad",
            "TaggedAddresses": null,
            "Tags": null,
            "TaskName": ""
        }
    },
    {
        "op": "add",
        "path": "/TaskGroups/0/Networks/0/DynamicPorts/1",
        "value": {
            "HostNetwork": "default",
            "Label": "auth",
            "To": 4180,
            "Value": 0
        }
    }
}
}
