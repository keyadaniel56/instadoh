import client from './client'

export const authApi = {
  login(email, password) {
    return client.post('/v1/auth/login', { email, password })
  },

  register(data) {
    return client.post('/v1/auth/register', data)
  },

  refreshToken() {
    return client.post('/v1/auth/refresh')
  },

  getProfile() {
    return client.get('/v1/users/me')
  },

  getCountries() {
    return client.get('/v1/countries')
  },
}