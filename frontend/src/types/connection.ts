export type Protocol = 'smb' | 'webdav' | 'ftp'

export interface ConnectionInput {
  id?: string
  name: string
  protocol: Protocol
  host?: string
  port?: number
  baseUrl?: string
  share?: string
  rootPath?: string
  username?: string
  domain?: string
  passiveMode?: boolean
  password?: string
  savePassword?: boolean
}

export interface ConnectionProfile {
  id: string
  name: string
  protocol: Protocol
  host?: string
  port?: number
  baseUrl?: string
  share?: string
  rootPath?: string
  username?: string
  domain?: string
  passiveMode?: boolean
  credentialKey?: string
  passwordSaved: boolean
  createdAt: string
  updatedAt: string
}
