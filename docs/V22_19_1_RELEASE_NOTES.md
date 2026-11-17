# Nexora Log Intelligence v22.19.1-0

- Published template parsing emits KV fields alongside header fields. Valid source keys are preserved, including keys outside the Known Key Catalog.
- Use the workspace header's Favorite control to add a shortcut to the sidebar. The controller stores shortcuts for the signed-in user and role.
- Add a single key directly in Known Key Catalog. On a published parser configuration, create a revision draft to edit field names or KV aliases. The active revision stays in service until its replacement is approved and published.
- Endpoint Configuration shows client-reported Loki and ClickHouse delivery counts and provides approved rollout scopes for one test client, selected clients, or all online clients. The test client sends its own logs to the endpoint.
- Upgrade parser clients for flat output. Newly generated scoped flat parser packs require v22.19.1 because older clients can reject valid keys that contain dots or hyphens.
