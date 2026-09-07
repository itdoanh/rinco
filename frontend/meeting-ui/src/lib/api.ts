import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

// Signaling API
export const signalingApi = {
  async getRoomStatus(roomId: string) {
    const response = await api.get(`/api/signaling?room_id=${roomId}`);
    return response.data;
  },

  async sendSignalingMessage(action: string, roomId: string, userId: string, data: unknown) {
    const response = await api.post('/api/signaling', {
      action,
      room_id: roomId,
      user_id: userId,
      data,
    });
    return response.data;
  },
};

// Room API
export const roomApi = {
  async getRoom(roomId: string) {
    const response = await api.get(`/api/room?room_id=${roomId}`);
    return response.data;
  },

  async createRoom(roomId: string, name: string, maxParticipants = 10) {
    const response = await api.post('/api/room', {
      room_id: roomId,
      name,
      max_participants: maxParticipants,
    });
    return response.data;
  },

  async deleteRoom(roomId: string) {
    const response = await api.delete(`/api/room?room_id=${roomId}`);
    return response.data;
  },
};
