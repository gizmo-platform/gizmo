---
global:
  scrape_interval: 2s

scrape_configs:
  - job_name: prometheus
    static_configs:
      - targets: ["localhost:9090"]
  - job_name: gizmo
    static_configs:
      - targets: ["localhost:8080"]
