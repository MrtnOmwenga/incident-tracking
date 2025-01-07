import api from '@/config/axios';
import type { Image } from '@/types/incident';

export class ImageService {
  private static instance: ImageService;
  private readonly baseUrl = '/incidents';

  private constructor() {}

  public static get(): ImageService {
    if (!ImageService.instance) {
      ImageService.instance = new ImageService();
    }
    return ImageService.instance;
  }

  async getImagesForIncident(incidentId: string): Promise<Image[]> {
    const response = await api.get<Image[]>(`${this.baseUrl}/${incidentId}/images`);
    return response.data;
  }

  async uploadImage(incidentId: string, file: File): Promise<Image> {
    const formData = new FormData();
    formData.append('image', file);

    const response = await api.post<Image>(
      `${this.baseUrl}/${incidentId}/images`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    )
    return response.data;
  }

  async deleteImage(incidentId, imageId: string): Promise<void> {
    await api.delete(`${this.baseUrl}/${incidentId}/images/${imageId}`);
  }
}