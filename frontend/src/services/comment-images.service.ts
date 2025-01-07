import api from '@/config/axios';
import type { Image } from '@/types/incident';

export class CommentImageService {
  private static instance: CommentImageService;
  private readonly baseUrl = '/incidents';

  private constructor() {}

  public static get(): CommentImageService {
    if (!CommentImageService.instance) {
      CommentImageService.instance = new CommentImageService();
    }
    return CommentImageService.instance;
  }

  async getImagesForComment(commentId: string): Promise<Image[]> {
    const response = await api.get<Image[]>(`${this.baseUrl}/${commentId}/images`);
    return response.data;
  }

  async uploadCommentImage(commentId: string, file: File): Promise<Image> {
    const formData = new FormData();
    formData.append('image', file);

    const response = await api.post<Image>(
      `${this.baseUrl}/${commentId}/comments/images`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    )
    return response.data;
  }

  async deleteImage(commentId, imageId: string): Promise<void> {
    await api.delete(`${this.baseUrl}/${commentId}/comments/images/${imageId}`);
  }
}