// Authentication mode is supplied by application composition, without importing stores.
export type AuthRuntimeConfig = {
  authDisabled: boolean
  httpRemoteUserEnabled: boolean
  configLoadSucceeded: boolean
}
let readConfig: () => AuthRuntimeConfig = () => ({
  authDisabled: false,
  httpRemoteUserEnabled: false,
  configLoadSucceeded: false,
})
export const authRuntime = { readConfig: () => readConfig() }
export function configureAuthRuntime(reader: () => AuthRuntimeConfig): void {
  readConfig = reader
}
