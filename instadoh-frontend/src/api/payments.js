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

  getRates(currency = 'USD') {
    return client.get('/v1/payments/rates', {
      params: { currency },
    })
  },

  convertCurrency(data) {
    return client.post('/v1/payments/convert', data)
  },
}