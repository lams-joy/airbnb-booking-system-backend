"use client";

import { useState, useEffect } from 'react';
import dynamic from 'next/dynamic';
import { Search, MapPin, Calendar, DollarSign, Users, CheckCircle, Clock, AlertCircle } from 'lucide-react';
import { api, Property } from '@/lib/api';

// Dynamic import for Leaflet (Client side only)
const Map = dynamic(() => import('@/components/Map'), { ssr: false });

export default function Home() {
  const [properties, setProperties] = useState<Property[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchParams, setSearchParams] = useState({
    lat: '37.7749', // Default San Francisco
    lng: '-122.4194',
    radius_km: '20',
  });

  const [selectedProperty, setSelectedProperty] = useState<Property | null>(null);
  const [bookingDates, setBookingDates] = useState({
    check_in: '2026-09-01',
    check_out: '2026-09-05',
    guest_id: 'guest_user_101',
  });

  const [holdStatus, setHoldStatus] = useState<string | null>(null);
  const [bookingStatus, setBookingStatus] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const fetchProperties = async () => {
    setLoading(true);
    setErrorMessage(null);
    try {
      const res = await api.get<Property[]>('/api/properties/search', {
        params: searchParams,
      });
      setProperties(res.data);
    } catch (err: any) {
      setErrorMessage(err.response?.data || 'Failed to fetch properties');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProperties();
  }, []);

  const handleHold = async () => {
    if (!selectedProperty) return;
    setErrorMessage(null);
    setHoldStatus(null);
    try {
      const res = await api.post('/api/bookings/hold', {
        property_id: selectedProperty.id,
        guest_id: bookingDates.guest_id,
        check_in: bookingDates.check_in,
        check_out: bookingDates.check_out,
      });
      setHoldStatus(`? ${res.data.message} (Hold key active in Redis)`);
    } catch (err: any) {
      setErrorMessage(err.response?.data || 'Hold failed');
    }
  };

  const handleConfirmBooking = async () => {
    if (!selectedProperty) return;
    setErrorMessage(null);
    setBookingStatus(null);

    const nights = 4;
    const totalPrice = selectedProperty.price_per_night * nights;

    try {
      const res = await api.post('/api/bookings/confirm', {
        property_id: selectedProperty.id,
        guest_id: bookingDates.guest_id,
        check_in: bookingDates.check_in,
        check_out: bookingDates.check_out,
        total_price: totalPrice,
      });
      setBookingStatus(`?? Booking Confirmed! ID: ${res.data.booking_id}`);
      setHoldStatus(null);
    } catch (err: any) {
      setErrorMessage(err.response?.data || 'Booking confirmation failed');
    }
  };

  const mapCenter: [number, number] = [
    parseFloat(searchParams.lat) || 37.7749,
    parseFloat(searchParams.lng) || -122.4194,
  ];

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col font-sans text-gray-900">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between sticky top-0 z-50">
        <div className="flex items-center space-x-2">
          <div className="w-8 h-8 bg-rose-500 rounded-lg flex items-center justify-center text-white font-bold text-lg">
            A
          </div>
          <span className="text-xl font-bold tracking-tight text-rose-500">Airbnb Architecture</span>
        </div>

        {/* Search Controls */}
        <div className="flex items-center bg-gray-100 p-1.5 rounded-full border border-gray-200 shadow-sm space-x-2 text-sm">
          <div className="flex items-center px-3 space-x-1.5 border-r border-gray-300">
            <MapPin className="w-4 h-4 text-gray-500" />
            <input
              type="text"
              placeholder="Lat"
              value={searchParams.lat}
              onChange={(e) => setSearchParams({ ...searchParams, lat: e.target.value })}
              className="w-16 bg-transparent focus:outline-none text-xs font-mono"
            />
            <input
              type="text"
              placeholder="Lng"
              value={searchParams.lng}
              onChange={(e) => setSearchParams({ ...searchParams, lng: e.target.value })}
              className="w-16 bg-transparent focus:outline-none text-xs font-mono"
            />
          </div>

          <div className="flex items-center px-3 space-x-1">
            <span className="text-xs text-gray-500">Radius:</span>
            <input
              type="number"
              value={searchParams.radius_km}
              onChange={(e) => setSearchParams({ ...searchParams, radius_km: e.target.value })}
              className="w-12 bg-transparent focus:outline-none text-xs font-medium"
            />
            <span className="text-xs text-gray-500">km</span>
          </div>

          <button
            onClick={fetchProperties}
            disabled={loading}
            className="bg-rose-500 text-white p-2 rounded-full hover:bg-rose-600 transition"
          >
            <Search className="w-4 h-4" />
          </button>
        </div>
      </header>

      {/* Main Container */}
      <main className="flex-1 grid grid-cols-1 lg:grid-cols-12 gap-4 p-4 h-[calc(100vh-80px)]">
        {/* Left Side: Property Grid */}
        <div className="lg:col-span-7 overflow-y-auto space-y-4 pr-2">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-800">
              {properties.length} Properties within {searchParams.radius_km}km
            </h2>
            {loading && <span className="text-sm text-rose-500 animate-pulse">Querying PostGIS...</span>}
          </div>

          {errorMessage && (
            <div className="bg-rose-50 border border-rose-200 text-rose-700 px-4 py-3 rounded-lg text-sm flex items-center space-x-2">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{errorMessage}</span>
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {properties.map((prop) => (
              <div
                key={prop.id}
                onClick={() => setSelectedProperty(prop)}
                className={`bg-white rounded-xl border p-4 shadow-sm hover:shadow-md transition cursor-pointer ${
                  selectedProperty?.id === prop.id ? 'border-rose-500 ring-2 ring-rose-100' : 'border-gray-200'
                }`}
              >
                <div className="flex justify-between items-start">
                  <h3 className="font-semibold text-gray-900 truncate">{prop.title}</h3>
                  <span className="text-sm font-bold text-gray-900">${prop.price_per_night}<span className="text-xs text-gray-500 font-normal">/night</span></span>
                </div>
                <p className="text-xs text-gray-500 mt-1 line-clamp-2">{prop.description}</p>
                <div className="mt-3 flex items-center justify-between text-xs text-gray-500">
                  <span className="flex items-center"><Users className="w-3.5 h-3.5 mr-1" /> Max {prop.max_guests} guests</span>
                  {prop.distance_meters !== undefined && (
                    <span className="bg-gray-100 px-2 py-0.5 rounded-full font-mono">
                      {(prop.distance_meters / 1000).toFixed(2)} km away
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Right Side: Map + Booking Panel */}
        <div className="lg:col-span-5 flex flex-col space-y-4">
          <div className="flex-1 min-h-[300px] border border-gray-200 rounded-xl overflow-hidden bg-gray-200 shadow-sm relative">
            <Map
              properties={properties}
              center={mapCenter}
              onSelectProperty={(prop) => setSelectedProperty(prop)}
            />
          </div>

          {/* Booking Drawer */}
          {selectedProperty ? (
            <div className="bg-white border border-gray-200 rounded-xl p-4 shadow-sm space-y-3">
              <div className="flex justify-between items-start border-b border-gray-100 pb-2">
                <div>
                  <h3 className="font-semibold text-sm text-gray-900">{selectedProperty.title}</h3>
                  <p className="text-xs text-gray-500">${selectedProperty.price_per_night} per night</p>
                </div>
                <button
                  onClick={() => setSelectedProperty(null)}
                  className="text-xs text-gray-400 hover:text-gray-600"
                >
                  Close
                </button>
              </div>

              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <label className="block text-gray-500 mb-1">Check-in</label>
                  <input
                    type="date"
                    value={bookingDates.check_in}
                    onChange={(e) => setBookingDates({ ...bookingDates, check_in: e.target.value })}
                    className="w-full border border-gray-300 rounded px-2 py-1"
                  />
                </div>
                <div>
                  <label className="block text-gray-500 mb-1">Check-out</label>
                  <input
                    type="date"
                    value={bookingDates.check_out}
                    onChange={(e) => setBookingDates({ ...bookingDates, check_out: e.target.value })}
                    className="w-full border border-gray-300 rounded px-2 py-1"
                  />
                </div>
              </div>

              {holdStatus && (
                <div className="bg-emerald-50 text-emerald-800 text-xs p-2 rounded border border-emerald-200 flex items-center space-x-1">
                  <Clock className="w-3.5 h-3.5 text-emerald-600" />
                  <span>{holdStatus}</span>
                </div>
              )}

              {bookingStatus && (
                <div className="bg-blue-50 text-blue-800 text-xs p-2 rounded border border-blue-200 flex items-center space-x-1">
                  <CheckCircle className="w-3.5 h-3.5 text-blue-600" />
                  <span>{bookingStatus}</span>
                </div>
              )}

              <div className="flex space-x-2 pt-1">
                <button
                  onClick={handleHold}
                  className="flex-1 bg-amber-500 hover:bg-amber-600 text-white text-xs py-2 rounded-lg font-medium transition"
                >
                  Hold for 10 Mins (Redis)
                </button>
                <button
                  onClick={handleConfirmBooking}
                  className="flex-1 bg-rose-500 hover:bg-rose-600 text-white text-xs py-2 rounded-lg font-medium transition"
                >
                  Confirm (PostgreSQL)
                </button>
              </div>
            </div>
          ) : (
            <div className="bg-white border border-gray-200 rounded-xl p-6 text-center text-xs text-gray-500 shadow-sm">
              Select a property pin on the map or click a card to start booking.
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
