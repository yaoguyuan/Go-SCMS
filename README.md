# Go-SCMS

![Gin](https://img.shields.io/badge/Gin-v1.10.0-blue)
![Gorm](https://img.shields.io/badge/Gorm-v1.25.12-green)
![JWT](https://img.shields.io/badge/JWT-v5.2.2-red)
![Casbin](https://img.shields.io/badge/Casbin-v2.105.0-orange)

English | [简体中文](#简介)

## Introduction

This is a basic golang project that implements a content management system with several security features.

- **Authentication**: Using JWT for identity authentication
- **Authorization**: Using Casbin for API level access control
- **Audit**: Recording significant operations for log audit
- To be expanded...

## Requirements

- Go 1.24+
- MySQL 5.7+

## Quick Start

```bash
git clone https://github.com/yaoguyuan/Go-SCMS.git

cd Go-SCMS-main

go mod tidy

# Configure the environment variables for the application in '.env'.

go run main.go
```

---

# Go-SCMS

![Gin](https://img.shields.io/badge/Gin-v1.10.0-blue)
![Gorm](https://img.shields.io/badge/Gorm-v1.25.12-green)
![JWT](https://img.shields.io/badge/JWT-v5.2.2-red)
![Casbin](https://img.shields.io/badge/Casbin-v2.105.0-orange)

[English](#introduction) | 简体中文

## 简介

这是一个基础的 Golang 项目，实现了一个内容管理系统，具有多种安全特性。

- **身份认证**：使用 JWT 进行身份验证
- **授权控制**：使用 Casbin 进行 API 级别的访问控制
- **审计日志**：记录重要操作以便日志审计
- 更多功能开发中...

## 环境需求

- Go 1.24+
- MySQL 5.7+

## 快速开始

```bash
git clone https://github.com/yaoguyuan/Go-SCMS.git

cd Go-SCMS-main

go mod tidy

# 在'.env'文件中配置应用程序的环境变量

go run main.go
```
