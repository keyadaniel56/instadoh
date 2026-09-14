import client from './client'

export const paymentsApi = {
  getBalance() {
    return client.get('/v1/payments/balance')
  },

  getStats() {
    return client.get('/v1/payments/stats')
  },

  listTransactions(page = 1, limit = 20) {
    return client.get('/v1/payments/transactions', {
      params: { page, limit },
    })
  },

  getTransaction(id) {
    return client.get(`/v1/payments/transactions/${id}`)
  },

  createInvoice(data) {
    return client.post('/v1/payments/invoices', data)
  },

  sendPayment(data) {
    return client.post('/v1/payments/send', data)
  },

  sendByEmail(data) {
    return client.post('/v1/payments/send-by-email', data)
  },

  withdraw(data) {
    return client.post('/v1/payments/withdraw', data)
  },

  resolveRecipient(email) {
    return client.get('/v1/payments/resolve', {
      params: { email },
    })
  },

  getConfig() {
    return client.get('/v1/config')
  },

  getRates(currency = 'USD') {
    return client.get('/v1/payments/rates', {
      params: { currency },
    })
  },

  convertCurrency(data) {
    return client.post('/v1/payments/convert', data)
  },
}