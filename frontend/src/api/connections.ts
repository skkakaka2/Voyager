import {
  DeleteConnection,
  ListConnections,
  SaveConnection,
  TestConnection
} from '../../wailsjs/go/main/App'
import type { ConnectionInput, ConnectionProfile } from '../types/connection'

export function listConnections(): Promise<ConnectionProfile[]> {
  return ListConnections() as Promise<ConnectionProfile[]>
}

export function saveConnection(input: ConnectionInput): Promise<ConnectionProfile> {
  return SaveConnection(input) as Promise<ConnectionProfile>
}

export function deleteConnection(id: string): Promise<void> {
  return DeleteConnection(id)
}

export function testConnection(input: ConnectionInput): Promise<void> {
  return TestConnection(input)
}
