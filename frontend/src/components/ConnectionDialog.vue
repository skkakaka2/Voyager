<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import type { ConnectionInput, ConnectionProfile, Protocol } from '../types/connection'

const props = defineProps<{
  modelValue: boolean
  connection: ConnectionProfile | null
  submitting: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'save', value: ConnectionInput): void
}>()

const form = reactive<ConnectionInput>({
  name: '',
  protocol: 'webdav',
  host: '',
  port: undefined,
  baseUrl: '',
  share: '',
  rootPath: '',
  username: '',
  domain: '',
  passiveMode: true,
  password: '',
  savePassword: true
})

const title = computed(() => (props.connection ? '编辑连接' : '新增连接'))

watch(
  () => [props.modelValue, props.connection] as const,
  ([visible, connection]) => {
    if (!visible) {
      return
    }
    Object.assign(form, {
      id: connection?.id,
      name: connection?.name ?? '',
      protocol: connection?.protocol ?? 'webdav',
      host: connection?.host ?? '',
      port: connection?.port,
      baseUrl: connection?.baseUrl ?? '',
      share: connection?.share ?? '',
      rootPath: connection?.rootPath ?? '',
      username: connection?.username ?? '',
      domain: connection?.domain ?? '',
      passiveMode: connection?.passiveMode ?? true,
      password: '',
      savePassword: connection?.passwordSaved ?? true
    })
  },
  { immediate: true }
)

function close() {
  emit('update:modelValue', false)
}

function submit() {
  emit('save', { ...form, protocol: form.protocol as Protocol })
}
</script>

<template>
  <el-dialog :model-value="modelValue" :title="title" width="560px" @close="close">
    <el-form label-position="top" @submit.prevent="submit">
      <el-form-item label="连接名称" required>
        <el-input v-model="form.name" placeholder="例如：办公室 WebDAV" />
      </el-form-item>

      <el-form-item label="协议" required>
        <el-segmented
          v-model="form.protocol"
          :options="[
            { label: 'WebDAV', value: 'webdav' },
            { label: 'FTP', value: 'ftp' },
            { label: 'SMB', value: 'smb' }
          ]"
        />
      </el-form-item>

      <template v-if="form.protocol === 'webdav'">
        <el-form-item label="Base URL" required>
          <el-input v-model="form.baseUrl" placeholder="https://example.com/dav" />
        </el-form-item>
      </template>

      <template v-else>
        <el-form-item label="主机" required>
          <el-input v-model="form.host" placeholder="files.example.com" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" />
        </el-form-item>
      </template>

      <el-form-item v-if="form.protocol === 'smb'" label="共享名">
        <el-input v-model="form.share" placeholder="留空则浏览服务器共享列表" />
      </el-form-item>

      <el-form-item label="远端根路径">
        <el-input v-model="form.rootPath" placeholder="/" />
      </el-form-item>

      <el-form-item label="用户名">
        <el-input v-model="form.username" autocomplete="username" />
      </el-form-item>

      <el-form-item v-if="form.protocol === 'smb'" label="域">
        <el-input v-model="form.domain" placeholder="可选" />
      </el-form-item>

      <el-form-item label="密码">
        <el-input
          v-model="form.password"
          autocomplete="current-password"
          placeholder="留空则保持当前密码设置"
          show-password
          type="password"
        />
      </el-form-item>

      <el-form-item>
        <el-checkbox v-model="form.savePassword">保存到系统钥匙串</el-checkbox>
      </el-form-item>

      <el-alert
        v-if="form.protocol === 'ftp'"
        :closable="false"
        show-icon
        title="普通 FTP 为明文传输，请只在可信网络中使用。"
        type="warning"
      />
      <el-alert
        v-if="form.protocol === 'webdav' && form.baseUrl?.startsWith('http://')"
        :closable="false"
        class="security-alert"
        show-icon
        title="HTTP WebDAV 为明文传输，建议优先使用 HTTPS。"
        type="warning"
      />
    </el-form>

    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button :loading="submitting" type="primary" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.security-alert {
  margin-top: 8px;
}
</style>
