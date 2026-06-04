<script setup lang="ts">
defineProps<{
  title: string
  description: string
}>()
</script>

<template>
  <div class="empty-state">
    <el-empty :description="description">
      <template #default>
        <h2>{{ title }}</h2>
      </template>
    </el-empty>
  </div>
</template>

<style scoped>
.empty-state {
  display: grid;
  min-height: 360px;
  place-items: center;
  border: 1px dashed #dcdfe6;
  border-radius: 6px;
  background: #ffffff;
}

.empty-state h2 {
  margin: 0 0 8px;
  color: #303133;
  font-size: 18px;
}
</style>
