job "my-app" {
    group "app" {
        task "app" {
            config { image = "my-app:1.0.0" }
        }
    }
}
job "my-app" {
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
        }
    }
}
job "${job_name}" {
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
        }
    }
}
job "${job_name}" {
    datacenters = ${jsonencode(datacenters)}
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
        }
    }
}
job "${job_name}" {
    datacenters = ${jsonencode(datacenters)}
    meta { owner =  "foo-departement" }
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
        }
    }
}
job "${job_name}" {
    datacenters = ${jsonencode(datacenters)}
    meta { owner =  "foo-departement" }
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
            vault  { policies = ${jsonencode(vault_policies)} }
        }
    }
}
job "${job_name}" {
    datacenters = ${jsonencode(datacenters)}
    meta { owner =  "foo-departement" }
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
%{if length(vault_policies)} vault  { policies = ${jsonencode(vault_policies)} } %{endif}
        }
    }
}
job "${job_name}" {
    datacenters = ${jsonencode(datacenters)}
    meta { owner =  "foo-departement" }
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
%{if length(vault_policies)} vault  { policies = ${jsonencode(vault_policies)} } %{endif}
            template {
               data = <<-EOF
               {{ with secret "database/creds/${job_name}" }}
               PGUSERNAME={{ .Data.username }}
               PGPASSWORD={{ .Data.password }}
               {{ end }}
               EOF
    	    }
        }
    }
}
job "${job_name}" {
    datacenters = ${jsonencode(datacenters)}
    meta { owner =  "${team_email}" }
    group "app" {
        task "app" {
            config { image = "my-app:${app_version}" }
%{if length(vault_policies)} vault  { policies = ${jsonencode(vault_policies)} } %{endif}
            template {
               data = <<-EOF
               {{ with secret "database/creds/${job_name}" }}
               PGUSERNAME={{ .Data.username }}
               PGPASSWORD={{ .Data.password }}
               {{ end }}
               EOF
    	    }
        }
        ${logshipper_sidecar_task}
    }
}
