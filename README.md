# Intelligent IoT Configuration Management System (Prototype)

This repository contains a prototype implementation of an **intelligent configuration management system for IoT devices in 5G networks**.

The system is developed as part of the research work:

**"Development of a Prototype Intelligent Configuration Management System for IoT Devices in 5G Telecommunication Networks"**

The project implements a microservice-based architecture that supports:

- configuration management for IoT devices
- controlled configuration deployment
- telemetry collection
- QoS prediction using machine learning

The prototype demonstrates how configuration rollout decisions can be supported by **predictive analytics and intelligent orchestration mechanisms**.

---

# System Architecture

The system is organized as a set of cooperating microservices.

Core components:

Device Simulator
↓
MQTT Broker (Mosquitto)
↓
Telemetry Ingestor

Device Registry ← REST → Configuration Service
↓
Deployment Orchestrator
↓
MQTT Adapter
↓
IoT Devices

Telemetry → PostgreSQL
Configurations → MongoDB

ML Service (QoS Prediction)


The architecture includes the following layers:

### Control Plane

Responsible for device and configuration management.

Services:

- `device-registry`
- `config-service`

Responsibilities:

- device registration
- configuration template management
- configuration versioning

---

### Delivery Layer

Responsible for delivering configuration updates to IoT devices.

Services:

- `mqtt-adapter`
- `device-simulator`

Responsibilities:

- publishing desired configuration
- receiving device acknowledgements
- tracking configuration application status

---

### Deployment Orchestration

Responsible for safe configuration rollout.

Service:

- `deployment-orchestrator`

Features:

- canary deployment
- failure rate monitoring
- automatic rollback decisions

---

### Telemetry Pipeline

Responsible for collecting device telemetry.

Service:

- `telemetry-ingestor`

Telemetry data includes:

- latency
- packet loss
- jitter
- RSSI
- battery level

All telemetry is stored in:


PostgreSQL → telemetry.metrics_raw


---

### Intelligence Layer

Responsible for predictive analytics.

Service:

- `ml-service`

Features:

- LSTM-based QoS degradation prediction
- telemetry time window analysis
- risk score estimation

Example response:


{
"deviceId": "...",
"riskScore": 0.013,
"riskLevel": "LOW"
}


---

# Technologies

The prototype uses the following technologies:

Backend services

- Go (Gin)
- Python (FastAPI)
- PyTorch (ML model)

Data storage

- PostgreSQL
- MongoDB
- Redis

Messaging

- MQTT (Mosquitto)
- NATS

Infrastructure

- Docker
- Docker Compose

---

# Implemented Features

The current MVP prototype includes:

Device management

- device registration
- device metadata storage

Configuration management

- configuration templates
- versioned configuration payloads

Configuration deployment

- MQTT configuration delivery
- device acknowledgement processing

Safe rollout mechanisms

- canary deployment
- automatic deployment monitoring
- failure-based rollback

Telemetry collection

- real-time telemetry ingestion
- time-series storage

Machine learning prediction

- LSTM QoS degradation prediction
- risk scoring based on telemetry data

---

# Example Workflow

Typical system operation:

1. Register IoT device
2. Create configuration template
3. Create configuration version
4. Deploy configuration using canary strategy
5. Devices apply configuration and send ACK
6. Telemetry is continuously collected
7. ML service predicts QoS degradation risk
8. Deployment decisions can be adjusted based on prediction

---

# Repository Structure


services/

device-registry
config-service
mqtt-adapter
deployment-orchestrator
telemetry-ingestor
ml-service
device-simulator

---

# Research Context

This project serves as the practical prototype for research on:

**Intelligent configuration management systems for IoT devices in 5G networks**

Key research topics addressed:

- configuration orchestration
- safe rollout strategies
- telemetry-based analytics
- machine learning for QoS prediction
- intelligent configuration validation

---

# Current Status

Implemented:

- configuration management
- MQTT configuration delivery
- canary deployment orchestration
- telemetry ingestion
- LSTM-based QoS prediction

Planned improvements:

- Digital Twin module for configuration validation
- integration of ML predictions into deployment decisions
- experimental evaluation scenarios

---

# Running the System

Start infrastructure:


docker compose -f docker-compose.yml -f docker-compose.services.yml up -d

---

# Research Demonstration

The prototype allows demonstration of:

- controlled configuration deployment
- telemetry-driven system monitoring
- machine learning-based QoS prediction
- intelligent configuration management workflows

This system serves as an experimental platform for evaluating intelligent configuration deployment strategies in IoT environments.