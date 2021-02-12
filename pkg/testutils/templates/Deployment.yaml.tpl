apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "17"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"qontract.caller_name":"saas-github-mirror","qontract.integration":"openshift-saas-deploy","qontract.integration_version":"0.1.0","qontract.sha256sum":"4af0efb326680a464e1f2c8f2780e36f749493e8f91cba1a7343e2508c30c08f","qontract.update":"2020-09-02T12:36:35"},"labels":{"app":"github-mirror"},"name":"github-mirror","namespace":"github-mirror-production"},"spec":{"replicas":5,"selector":{"matchLabels":{"app":"github-mirror"}},"strategy":{"rollingUpdate":{"maxSurge":1,"maxUnavailable":0},"type":"RollingUpdate"},"template":{"metadata":{"labels":{"app":"github-mirror"}},"spec":{"affinity":{"podAntiAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"podAffinityTerm":{"labelSelector":{"matchExpressions":[{"key":"app","operator":"In","values":["github-mirror"]}]},"topologyKey":"kubernetes.io/hostname"},"weight":100},{"podAffinityTerm":{"labelSelector":{"matchExpressions":[{"key":"app","operator":"In","values":["github-mirror"]}]},"topologyKey":"failure-domain.beta.kubernetes.io/zone"},"weight":100}]}},"containers":[{"env":[{"name":"GITHUB_USERS","value":"app-sre-bot:cs-sre-bot"},{"name":"GITHUB_MIRROR_URL","value":"https://github-mirror.devshift.net"},{"name":"CACHE_TYPE","value":"redis"},{"name":"PRIMARY_ENDPOINT","valueFrom":{"secretKeyRef":{"key":"db.endpoint","name":"ghmirror-elasticache-production"}}},{"name":"READER_ENDPOINT","value":"replica.ghmirror-redis-production.huo5rn.use1.cache.amazonaws.com"},{"name":"REDIS_PORT","valueFrom":{"secretKeyRef":{"key":"db.port","name":"ghmirror-elasticache-production"}}},{"name":"REDIS_TOKEN","valueFrom":{"secretKeyRef":{"key":"db.auth_token","name":"ghmirror-elasticache-production"}}},{"name":"REDIS_SSL","value":"True"}],"image":"quay.io/app-sre/github-mirror:5344cbb","imagePullPolicy":"Always","livenessProbe":{"httpGet":{"path":"/healthz","port":8080},"initialDelaySeconds":30,"periodSeconds":10,"timeoutSeconds":3},"name":"github-mirror","ports":[{"containerPort":8080,"name":"github-mirror"}],"readinessProbe":{"httpGet":{"path":"/healthz","port":8080},"initialDelaySeconds":3,"periodSeconds":10,"timeoutSeconds":3},"resources":{"limits":{"cpu":"1000m","memory":"1Gi"},"requests":{"cpu":"500m","memory":"800Mi"}}}]}}}}
    qontract.caller_name: saas-github-mirror
    qontract.integration: openshift-saas-deploy
    qontract.integration_version: 0.1.0
    qontract.sha256sum: 4af0efb326680a464e1f2c8f2780e36f749493e8f91cba1a7343e2508c30c08f
    qontract.update: 2020-09-02T12:36:35
  creationTimestamp: "2020-02-10T15:01:35Z"
  generation: 8482
  labels:
    app: github-mirror
  name: github-mirror
  namespace: github-mirror-production
  resourceVersion: "267502440"
  selfLink: /apis/apps/v1/namespaces/github-mirror-production/deployments/github-mirror
  uid: 3b4d6091-4c16-11ea-bf75-023e213e25c3
spec:
  progressDeadlineSeconds: 2147483647
  replicas: {{ .Replicas }}
  revisionHistoryLimit: 2147483647
  selector:
    matchLabels:
      app: github-mirror
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: github-mirror
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - github-mirror
              topologyKey: kubernetes.io/hostname
            weight: 100
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - github-mirror
              topologyKey: failure-domain.beta.kubernetes.io/zone
            weight: 100
      containers:
      - env:
        - name: GITHUB_USERS
          value: app-sre-bot:cs-sre-bot
        - name: GITHUB_MIRROR_URL
          value: https://github-mirror.devshift.net
        - name: CACHE_TYPE
          value: redis
        - name: PRIMARY_ENDPOINT
          valueFrom:
            secretKeyRef:
              key: db.endpoint
              name: ghmirror-elasticache-production
        - name: READER_ENDPOINT
          value: replica.ghmirror-redis-production.huo5rn.use1.cache.amazonaws.com
        - name: REDIS_PORT
          valueFrom:
            secretKeyRef:
              key: db.port
              name: ghmirror-elasticache-production
        - name: REDIS_TOKEN
          valueFrom:
            secretKeyRef:
              key: db.auth_token
              name: ghmirror-elasticache-production
        - name: REDIS_SSL
          value: "True"
        image: quay.io/app-sre/github-mirror:5344cbb
        imagePullPolicy: Always
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 30
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 3
        name: github-mirror
        ports:
        - containerPort: 8080
          name: github-mirror
          protocol: TCP
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 3
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 3
        resources:
          {{- if .ResourceLimits }}
          limits:
            memory: 1Gi
          {{- end }}
          {{- if .ResourceRequests }}
          requests:
            cpu: 500m
            memory: 800Mi
          {{- end }}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: {{ .Replicas }}
  conditions:
  - lastTransitionTime: "2020-09-08T07:59:45Z"
    lastUpdateTime: "2020-09-08T07:59:45Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  observedGeneration: 8482
  readyReplicas: {{ .Replicas }}
  replicas: {{ .Replicas }}
  updatedReplicas: {{ .Replicas }}
