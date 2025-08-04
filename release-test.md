## 📋 Terraform Plan Summary

**Total resources affected:** 2

| Action | Count | Resources |
|--------|-------|----------|
| 🟡 **Update** | 1 | module.elasticache_redis.aws_elasticache_cluster.cluster |
| 🔄 **Replace** | 1 | module.elasticache_redis.aws_elasticache_parameter_group.default |

### 🟡 Resources to be Updated

#### `module.elasticache_redis.aws_elasticache_cluster.cluster`

**Attributes being modified:**

- **parameter_group_name**: "d1-redis-cache-params" *(removed)*

### 🔄 Resources to be Replaced

#### `module.elasticache_redis.aws_elasticache_parameter_group.default`

**Reason for replacement:** Attribute 'name' changed from 'd1-redis-cache-params' to 'd1-redis6.x-cache-params' (forces replacement)

**Attribute changes:**

- **tags**: {} → (null)
- **name**: "d1-redis-cache-params" → "d1-redis6.x-cache-params"

---
*Generated from Terraform 1.9.8 plan*
