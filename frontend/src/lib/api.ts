import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface Property {
  id: string;
  host_id: string;
  title: string;
  description: string;
  price_per_night: number;
  max_guests: number;
  latitude: number;
  longitude: number;
  distance_meters?: number;
}
