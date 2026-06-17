import client from './client'

export const crossBorderApi = {
  getQuote(fromCurrency, toCurrency, amount) {
    return client.get('/v1/cross-border/quote', {
      params: { from_currency: fromCurrency, to_currency: toCurrency, amount },
    })
  },

  sendCrossBorder(data) {
    return client.post('/v1/cross-border/send', data)
  },

  listTransactions(page = 1, limit = 20) {
    return client.get('/v1/cross-border/transactions', {
      params: { page, limit },
    })
  },

  getTransaction(id) {
    return client.get(`/v1/cross-border/transactions/${id}`)
  },
}

export const mpesaApi = {
  deposit(data) {
    return client.post('/v1/mpesa/deposit', data)
  },

  withdraw(data) {
    return client.post('/v1/mpesa/withdraw', data)
  },
}

export const ugandaMobileApi = {
  deposit(data) {
    return client.post('/v1/uganda-mobile/deposit', data)
  },

  withdraw(data) {
    return client.post('/v1/uganda-mobile/withdraw', data)
  },
}