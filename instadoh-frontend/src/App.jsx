import React from 'react'
import { Routes, Route } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Features from './components/Features'
import HowItWorks from './components/HowItWorks'
import Pricing from './components/Pricing'
import CTA from './components/CTA'
import Footer from './components/Footer'
import Dashboard from './components/Dashboard'
import Login from './components/Login'
import Signup from './components/Signup'
import ProtectedRoute from './components/ProtectedRoute'
import './App.css'

function LandingPage() {
  return (
    <>
      <Hero />
      <Features />
      <HowItWorks />
      <Pricing />
      <CTA />
    </>
  )
}

function App() {
  return (
    <AuthProvider>
      <div className="app">
        <Navbar />
        <Routes>
          <Route path="/" element={<LandingPage />} />
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<Signup />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
        </Routes>
        <Footer />
      </div>
    </AuthProvider>
  )
}

export default App
