{
  "__inputs": [],
  "__elements": {},
  "__requires": [
    {
      "type": "panel",
      "id": "dashlist",
      "name": "Dashboard list",
      "version": ""
    },
    {
      "type": "grafana",
      "id": "grafana",
      "name": "Grafana",
      "version": "10.1.5"
    },
    {
      "type": "panel",
      "id": "text",
      "name": "Text",
      "version": ""
    }
  ],
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": {
          "type": "grafana",
          "uid": "-- Grafana --"
        },
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "id": null,
  "links": [],
  "liveNow": false,
  "panels": [
    {
      "datasource": {
        "type": "datasource",
        "uid": "grafana"
      },
      "description": "",
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      },
      "id": 1,
      "options": {
        "code": {
          "language": "plaintext",
          "showLineNumbers": false,
          "showMiniMap": false
        },
        "content": "Welcome to the Gizmo Monitoring System.  You can find information here about robot status, system components, and use this information for troubleshooting systems on the fly.\n\nData is retrieved from remote systems every 2 seconds, but robots will only send data when powered on and connected to the FMS.",
        "mode": "markdown"
      },
      "pluginVersion": "10.1.5",
      "title": "Monitoring Home",
      "type": "text"
    },
    {
      "datasource": {
        "type": "datasource",
        "uid": "grafana"
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 12,
        "y": 0
      },
      "id": 2,
      "options": {
        "includeVars": false,
        "keepTime": false,
        "maxItems": 10,
        "query": "",
        "showHeadings": true,
        "showRecentlyViewed": false,
        "showSearch": true,
        "showStarred": false,
        "tags": []
      },
      "pluginVersion": "10.1.5",
      "title": "Dashboards",
      "type": "dashlist"
    }
  ],
  "refresh": "",
  "schemaVersion": 38,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": []
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "",
  "title": "Home",
  "uid": "a3cba7d1-d894-4537-8b91-d24d23a3a092",
  "version": 1,
  "weekStart": ""
}
