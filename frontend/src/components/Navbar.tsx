import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import api from '../api/axios'

export default function Navbar() {
  const navigate = useNavigate()
  const { user, logout } = useAuthStore()

  const handleLogout = async () => {
    const refreshToken = localStorage.getItem('refreshToken')
    try {
      await api.post('/auth/logout', { refreshToken })
    } catch {}
    logout()
    navigate('/login')
  }

  return (
    <nav className="bg-white border-b border-gray-200 px-6 py-4">
      <div className="max-w-6xl mx-auto flex items-center justify-between">

        {/* Logo */}
        <div
          className="flex items-center gap-3 cursor-pointer"
          onClick={() => navigate('/dashboard')}
        >
          <div className="w-9 h-9 bg-blue-600 rounded-xl flex items-center justify-center">
            <span className="text-white font-bold">B</span>
          </div>
          <span className="font-bold text-gray-800 text-lg">BankAPI</span>
        </div>

        {/* Links */}
        <div className="flex items-center gap-6">
          <button
            onClick={() => navigate('/dashboard')}
            className="text-gray-600 hover:text-blue-600 font-medium transition"
          >
            Dashboard
          </button>
          <button
            onClick={() => navigate('/accounts')}
            className="text-gray-600 hover:text-blue-600 font-medium transition"
          >
            Cuentas
          </button>
          <button
            onClick={() => navigate('/transfer')}
            className="text-gray-600 hover:text-blue-600 font-medium transition"
          >
            Transferir
          </button>
        </div>

        {/* Usuario */}
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-sm font-medium text-gray-800">
              {user?.firstName} {user?.lastName}
            </p>
            <p className="text-xs text-gray-500">{user?.email}</p>
          </div>
          <button
            onClick={handleLogout}
            className="bg-red-50 hover:bg-red-100 text-red-600 font-medium px-4 py-2 rounded-lg transition text-sm"
          >
            Salir
          </button>
        </div>

      </div>
    </nav>
  )
}