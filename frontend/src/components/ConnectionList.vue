<script setup lang="ts">
import { Connection, Delete, Edit, Plus } from '@element-plus/icons-vue'
import type { ConnectionProfile } from '../types/connection'

defineProps<{
  connections: ConnectionProfile[]
  currentId: string
  loading: boolean
}>()

defineEmits<{
  (event: 'create'): void
  (event: 'edit', value: ConnectionProfile): void
  (event: 'remove', value: ConnectionProfile): void
  (event: 'select', value: string): void
  (event: 'test', value: ConnectionProfile): void
}>()
</script>

<template>
  <section class="connection-list">
    <div class="connection-list__header">
      <div>
        <h1>Voyager</h1>
        <span>远端文件管理</span>
      </div>
      <el-tooltip content="新增连接" placement="bottom">
        <el-button :icon="Plus" circle type="primary" @click="$emit('create')" />
      </el-tooltip>
    </div>

    <el-skeleton v-if="loading" :rows="5" animated />
    <el-empty v-else-if="connections.length === 0" description="暂无连接" />
    <div v-else class="connection-list__items">
      <button
        v-for="connection in connections"
        :key="connection.id"
        :class="['connection-item', { 'connection-item--active': connection.id === currentId }]"
        type="button"
        @click="$emit('select', connection.id)"
      >
        <span class="connection-item__main">
          <strong>{{ connection.name }}</strong>
          <small>{{ connection.protocol.toUpperCase() }}</small>
        </span>
        <span class="connection-item__meta">
          <el-tag v-if="connection.passwordSaved" size="small" type="success">已保存凭据</el-tag>
          <el-tag v-else size="small" type="info">未保存凭据</el-tag>
        </span>
        <span class="connection-item__actions">
          <el-tooltip content="测试连接" placement="bottom">
            <el-button :icon="Connection" circle size="small" @click.stop="$emit('test', connection)" />
          </el-tooltip>
          <el-tooltip content="编辑" placement="bottom">
            <el-button :icon="Edit" circle size="small" @click.stop="$emit('edit', connection)" />
          </el-tooltip>
          <el-tooltip content="删除" placement="bottom">
            <el-button :icon="Delete" circle size="small" type="danger" @click.stop="$emit('remove', connection)" />
          </el-tooltip>
        </span>
      </button>
    </div>
  </section>
</template>

<style scoped>
.connection-list {
  display: flex;
  height: 100%;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
}

.connection-list__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.connection-list__header h1 {
  margin: 0;
  color: #1f2d3d;
  font-size: 20px;
  font-weight: 650;
}

.connection-list__header span {
  color: #606266;
  font-size: 13px;
}

.connection-list__items {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
}

.connection-item {
  display: grid;
  min-height: 92px;
  grid-template-columns: 1fr auto;
  gap: 10px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  background: #ffffff;
  padding: 12px;
  text-align: left;
  cursor: pointer;
}

.connection-item:hover,
.connection-item--active {
  border-color: #409eff;
  background: #ecf5ff;
}

.connection-item__main,
.connection-item__meta,
.connection-item__actions {
  display: flex;
  align-items: center;
}

.connection-item__main {
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.connection-item__main strong {
  max-width: 170px;
  overflow: hidden;
  color: #1f2d3d;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connection-item__main small {
  color: #909399;
}

.connection-item__actions {
  grid-column: 1 / -1;
  gap: 6px;
}
</style>
