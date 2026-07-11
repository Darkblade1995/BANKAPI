import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Navbar from '../components/Navbar'
import api from '../api/axios'

interface Account {
  id: number
  balance: number
  currency: string
}

export default function Transfer() {
  const navigate = useNavigate()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [form, setForm] = useState({
    fromAccountId: '',
    toAccountId: '',
    amount: '',
  })
  const [result, setResult] = useState<any>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetchAccounts()
  }, [])

  const fetchAccounts = async () => {
    try {
      const res = await api.get('/accounts')
      setAccounts(res.data.accounts)
      if (res.data.accounts.length > 0) {
        setForm((f) => ({ ...f, fromAccountId: res.data.accounts[0].id.toString() }))
      }
    } catch (err) {
      console.error(err)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setResult(null)

    try {
      const res = await api.post('/transfers', {
        fromAccountId: parseInt(form.fromAccountId),
        toAccountId: parseInt(form.toAccountId),
        amount: parseInt(form.amount),
      })
      setResult(res.data)
      fetchAccounts()
    } catch (err: any) {
      setError(err.response?.data?.error || 'Error al transferir')
    } finally {
      setLoading(false)
    }
  }

  const formatBalance = (amount: number, currency: string) => {
    return new Intl.NumberFormat('es-CO', {
      style: 'currency',
      currency,
      minimumFractionDigits: 0,
    }).format(amount / 100)
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />

      <div className="max-w-2xl mx-auto px-6 py-8">

        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-800">Transferir</h1>
          <p className="text-gray-500 mt-1">Envía dinero entre cuentas</p>
        </div>

        <div className="bg-white rounded-2xl shadow-sm p-8">

          {/* Error */}
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-600 rounded-xl p-4 mb-6 text-sm">
              ❌ {error}
            </div>
          )}

          {/* Resultado exitoso */}
          {result && (
            <div className="bg-green-50 border border-green-200 rounded-xl p-6 mb-6">
              <p className="text-green-700 font-bold text-lg mb-3">
                ✅ Transferencia exitosa
              </p>
              <div className="space-y-2 text-sm text-green-600">
                <div className="flex justify-between">
                  <span>Monto enviado:</span>
                  <span className="font-bold">{result.amount} {result.fromCurrency}</span>
                </div>
                <div className="flex justify-between">
                  <span>Monto recibido:</span>
                  <span className="font-bold">{result.convertedAmount} {result.toCurrency}</span>
                </div>
                {result.fromCurrency !== result.toCurrency && (
                  <div className="flex justify-between">
                    <span>Tasa de cambio:</span>
                    <span className="font-bold">{result.exchangeRate}</span>
                  </div>
                )}
              </div>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">

            {/* Cuenta origen */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Cuenta origen
              </label>
              <select
                value={form.fromAccountId}
                onChange={(e) => setForm({ ...form, fromAccountId: e.target.value })}
                className="w-full border border-gray-300 rounded-xl px-4 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              >
                {accounts.map((account) => (
                  <option key={account.id} value={account.id}>
                    Cuenta #{account.id} — {formatBalance(account.balance, account.currency)} {account.currency}
                  </option>
                ))}
              </select>
            </div>

            {/* Cuenta destino */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                ID de cuenta destino
              </label>
              <input
                type="number"
                value={form.toAccountId}
                onChange={(e) => setForm({ ...form, toAccountId: e.target.value })}
                className="w-full border border-gray-300 rounded-xl px-4 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Ej: 2"
                required
              />
              <p className="text-xs text-gray-400 mt-1">
                Ingresa el ID de la cuenta a la que deseas transferir
              </p>
            </div>

            {/* Monto */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Monto (en centavos)
              </label>
              <input
                type="number"
                value={form.amount}
                onChange={(e) => setForm({ ...form, amount: e.target.value })}
                className="w-full border border-gray-300 rounded-xl px-4 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Ej: 100000 = $1.000"
                required
                min="1"
              />
              <p className="text-xs text-gray-400 mt-1">
                100000 centavos = $1.000 COP
              </p>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-4 rounded-xl transition disabled:opacity-50 text-lg"
            >
              {loading ? 'Procesando...' : '💸 Transferir'}
            </button>

          </form>
        </div>
      </div>
    </div>
  )
}