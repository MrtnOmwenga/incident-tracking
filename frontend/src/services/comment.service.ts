import api from '@/config/axios';
import type { 
  Comment, 
  CreateCommentDTO, 
} from '@/types/Comment';

export class CommentService {
  private static instance: CommentService;
  private readonly baseUrl = '/incidents';

  private constructor() {}

  public static get(): CommentService {
    if (!CommentService.instance) {
      CommentService.instance = new CommentService();
    }
    return CommentService.instance;
  }

  async getCommentById(commentId: string): Promise<Comment> {
    const response = await api.get<Comment>(`${this.baseUrl}/comments/${commentId}`);
    return response.data;
  }

  async getCommentForIncident(incidentId: string): Promise<Comment> {
    const response = await api.get<Comment>(`${this.baseUrl}/${incidentId}/comments`);
    return response.data;
  }

  async createComment(comment: CreateCommentDTO): Promise<Comment> {
    const response = await api.post<Comment>(`${this.baseUrl}/comments`, comment);
    return response.data;
  }

  async updateComment(id: string, comment: UpdateCommentDTO): Promise<Comment> {
    const response = await api.put<Comment>(`${this.baseUrl}/${id}/comments`, comment);
    return response.data;
  }

  async deleteComment(id: string): Promise<void> {
    await api.delete(`${this.baseUrl}/comments/${id}`);
  }
}
