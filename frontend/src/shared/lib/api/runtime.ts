// Authentication mode is supplied by application composition, without importing stores.
export type AuthRuntimeConfig = {
  httpRemoteUserEnabled: boolean
  configLoadSucceeded: boolean
}
let readConfig: () => AuthRuntimeConfig = () => ({
  httpRemoteUserEnabled: false,
  configLoadSucceeded: false,
})
export const authRuntime = { readConfig: () => readConfig() }
export function configureAuthRuntime(reader: () => AuthRuntimeConfig): void {
  readConfig = reader
}
