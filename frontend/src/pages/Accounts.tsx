import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Navbar from '../components/Navbar'
import api from '../api/axios'

interface Account {
  id: number
  userId: number
  balance: number
  currency: string
  createdAt: string
}

export default function Accounts() {
  const navigate = useNavigate()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [currency, setCurrency] = useState('COP')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    fetchAccounts()
  }, [])

  const fetchAccounts = async () => {
    try {
      const res = await api.get('/accounts')
      setAccounts(res.data.accounts)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const createAccount = async () => {
    setCreating(true)
    setError('')
    try {
      await api.post('/accounts', { currency })
      setShowCreate(false)
      fetchAccounts()
    } catch (err: any) {
      setError(err.response?.data?.error || 'Error al crear cuenta')
    } finally {
      setCreating(false)
    }
  }

  const formatBalance = (amount: number, currency: string) => {
    return new Intl.NumberFormat('es-CO', {
      style: 'currency',
      currency,
      minimumFractionDigits: 0,
    }).format(amount / 100)
  }

  const currencyColors: Record<string, string> = {
    COP: 'from-blue-600 to-blue-800',
    USD: 'from-green-600 to-green-800',
    EUR: 'from-purple-600 to-purple-800',
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />

      <div className="max-w-6xl mx-auto px-6 py-8">

        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-800">Mis Cuentas</h1>
            <p className="text-gray-500 mt-1">Administra tus cuentas bancarias</p>
          </div>
          <button
            onClick={() => setShowCreate(true)}
            className="bg-blue-600 hover:bg-blue-700 text-white font-semibold px-6 py-3 rounded-xl transition"
          >
            + Nueva cuenta
          </button>
        </div>

        {/* Modal crear cuenta */}
        {showCreate && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-white rounded-2xl p-8 w-full max-w-sm shadow-2xl">
              <h2 className="text-xl font-bold text-gray-800 mb-6">Nueva cuenta</h2>

              {error && (
                <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg p-3 mb-4 text-sm">
                  {error}
                </div>
              )}

              <div className="mb-6">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Moneda
                </label>
                <select
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="COP">🇨🇴 Peso colombiano (COP)</option>
                  <option value="USD">🇺🇸 Dólar americano (USD)</option>
                  <option value="EUR">🇪🇺 Euro (EUR)</option>
                </select>
              </div>

              <div className="flex gap-3">
                <button
                  onClick={() => setShowCreate(false)}
                  className="flex-1 border border-gray-300 text-gray-700 font-medium py-3 rounded-xl hover:bg-gray-50 transition"
                >
                  Cancelar
                </button>
                <button
                  onClick={createAccount}
                  disabled={creating}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 text-white font-medium py-3 rounded-xl transition disabled:opacity-50"
                >
                  {creating ? 'Creando...' : 'Crear'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Lista de cuentas */}
        {loading ? (
          <div className="text-center py-12 text-blue-600">Cargando...</div>
        ) : accounts.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-400 mb-4">No tienes cuentas aún</p>
            <button
              onClick={() => setShowCreate(true)}
              className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 transition"
            >
              Crear primera cuenta
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {accounts.map((account) => (
              <div
                key={account.id}
                onClick={() => navigate(`/accounts/${account.id}`)}
                className={`bg-gradient-to-br ${currencyColors[account.currency] || 'from-gray-600 to-gray-800'} rounded-2xl p-6 text-white cursor-pointer hover:shadow-xl transition`}
              >
                <div className="flex justify-between items-start mb-6">
                  <span className="text-white/70 text-sm">Cuenta #{account.id}</span>
                  <span className="bg-white/20 px-3 py-1 rounded-full text-xs font-bold">
                    {account.currency}
                  </span>
                </div>
                <p className="text-3xl font-bold mb-1">
                  {formatBalance(account.balance, account.currency)}
                </p>
                <p className="text-white/70 text-sm">Saldo disponible</p>
                <div className="mt-4 pt-4 border-t border-white/20">
                  <p className="text-white/60 text-xs">
                    Creada: {new Date(account.createdAt).toLocaleDateString('es-CO')}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}