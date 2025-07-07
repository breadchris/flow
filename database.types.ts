export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export type Database = {
  public: {
    Tables: {
      ai_recipe_generations: {
        Row: {
          additional_notes: string | null
          ai_model: string | null
          api_called_at: string | null
          api_completed_at: string | null
          api_request_id: string | null
          api_response_time_ms: number | null
          created_at: string | null
          difficulty: string | null
          error_message: string | null
          fallback_reason: string | null
          generated_recipe: Json | null
          generated_recipe_id: string | null
          generated_recipe_name: string | null
          generation_type: string | null
          has_images: boolean | null
          has_text_details: boolean | null
          id: string
          image_count: number | null
          image_urls: string[] | null
          images_processed: number | null
          processing_completed_at: string | null
          processing_time_ms: number | null
          recipe_name: string | null
          request_started_at: string | null
          session_id: string | null
          success: boolean | null
          total_images_submitted: number | null
          user_agent: string | null
          user_id: string | null
        }
        Insert: {
          additional_notes?: string | null
          ai_model?: string | null
          api_called_at?: string | null
          api_completed_at?: string | null
          api_request_id?: string | null
          api_response_time_ms?: number | null
          created_at?: string | null
          difficulty?: string | null
          error_message?: string | null
          fallback_reason?: string | null
          generated_recipe?: Json | null
          generated_recipe_id?: string | null
          generated_recipe_name?: string | null
          generation_type?: string | null
          has_images?: boolean | null
          has_text_details?: boolean | null
          id?: string
          image_count?: number | null
          image_urls?: string[] | null
          images_processed?: number | null
          processing_completed_at?: string | null
          processing_time_ms?: number | null
          recipe_name?: string | null
          request_started_at?: string | null
          session_id?: string | null
          success?: boolean | null
          total_images_submitted?: number | null
          user_agent?: string | null
          user_id?: string | null
        }
        Update: {
          additional_notes?: string | null
          ai_model?: string | null
          api_called_at?: string | null
          api_completed_at?: string | null
          api_request_id?: string | null
          api_response_time_ms?: number | null
          created_at?: string | null
          difficulty?: string | null
          error_message?: string | null
          fallback_reason?: string | null
          generated_recipe?: Json | null
          generated_recipe_id?: string | null
          generated_recipe_name?: string | null
          generation_type?: string | null
          has_images?: boolean | null
          has_text_details?: boolean | null
          id?: string
          image_count?: number | null
          image_urls?: string[] | null
          images_processed?: number | null
          processing_completed_at?: string | null
          processing_time_ms?: number | null
          recipe_name?: string | null
          request_started_at?: string | null
          session_id?: string | null
          success?: boolean | null
          total_images_submitted?: number | null
          user_agent?: string | null
          user_id?: string | null
        }
        Relationships: []
      }
      bread_photos: {
        Row: {
          bread_id: string
          bread_name: string
          comment: string | null
          created_at: string
          id: string
          likes_count: number | null
          photo_url: string
          rating: number | null
          user_id: string
        }
        Insert: {
          bread_id: string
          bread_name: string
          comment?: string | null
          created_at?: string
          id?: string
          likes_count?: number | null
          photo_url: string
          rating?: number | null
          user_id: string
        }
        Update: {
          bread_id?: string
          bread_name?: string
          comment?: string | null
          created_at?: string
          id?: string
          likes_count?: number | null
          photo_url?: string
          rating?: number | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "bread_photos_bread_id_fkey"
            columns: ["bread_id"]
            isOneToOne: false
            referencedRelation: "user_recipes"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "bread_photos_user_id_profiles_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      bread_progress: {
        Row: {
          bread_id: string
          bread_name: string
          completed_steps: string[] | null
          created_at: string
          current_step: number | null
          expected_finish_time: string
          id: string
          ingredient_notes: string | null
          is_completed: boolean | null
          last_updated: string
          start_time: string
          step_start_times: Json | null
          user_id: string
        }
        Insert: {
          bread_id: string
          bread_name: string
          completed_steps?: string[] | null
          created_at?: string
          current_step?: number | null
          expected_finish_time: string
          id?: string
          ingredient_notes?: string | null
          is_completed?: boolean | null
          last_updated?: string
          start_time: string
          step_start_times?: Json | null
          user_id: string
        }
        Update: {
          bread_id?: string
          bread_name?: string
          completed_steps?: string[] | null
          created_at?: string
          current_step?: number | null
          expected_finish_time?: string
          id?: string
          ingredient_notes?: string | null
          is_completed?: boolean | null
          last_updated?: string
          start_time?: string
          step_start_times?: Json | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "bread_progress_bread_id_fkey"
            columns: ["bread_id"]
            isOneToOne: false
            referencedRelation: "user_recipes"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "bread_progress_user_id_profiles_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      claude_doc_stars: {
        Row: {
          claude_doc_id: string
          created_at: string | null
          deleted_at: string | null
          id: string
          updated_at: string | null
          user_id: string
        }
        Insert: {
          claude_doc_id: string
          created_at?: string | null
          deleted_at?: string | null
          id: string
          updated_at?: string | null
          user_id: string
        }
        Update: {
          claude_doc_id?: string
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          updated_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_claude_doc_stars_claude_doc"
            columns: ["claude_doc_id"]
            isOneToOne: false
            referencedRelation: "claude_docs"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_claude_doc_stars_user"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      claude_doc_tags: {
        Row: {
          claude_doc_id: string
          created_at: string | null
          tag_id: string
        }
        Insert: {
          claude_doc_id: string
          created_at?: string | null
          tag_id: string
        }
        Update: {
          claude_doc_id?: string
          created_at?: string | null
          tag_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_claude_doc_tags_claude_doc"
            columns: ["claude_doc_id"]
            isOneToOne: false
            referencedRelation: "claude_docs"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_claude_doc_tags_tag"
            columns: ["tag_id"]
            isOneToOne: false
            referencedRelation: "tags"
            referencedColumns: ["id"]
          },
        ]
      }
      claude_docs: {
        Row: {
          content: string | null
          created_at: string | null
          deleted_at: string | null
          description: string | null
          downloads: number | null
          id: string
          is_public: boolean | null
          stars: number | null
          title: string
          updated_at: string | null
          user_id: string
          views: number | null
        }
        Insert: {
          content?: string | null
          created_at?: string | null
          deleted_at?: string | null
          description?: string | null
          downloads?: number | null
          id: string
          is_public?: boolean | null
          stars?: number | null
          title: string
          updated_at?: string | null
          user_id: string
          views?: number | null
        }
        Update: {
          content?: string | null
          created_at?: string | null
          deleted_at?: string | null
          description?: string | null
          downloads?: number | null
          id?: string
          is_public?: boolean | null
          stars?: number | null
          title?: string
          updated_at?: string | null
          user_id?: string
          views?: number | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_claude_docs_user"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      claude_sessions: {
        Row: {
          created_at: string | null
          deleted_at: string | null
          id: string
          messages: Json | null
          metadata: string | null
          session_id: string
          title: string | null
          updated_at: string | null
          user_id: string
        }
        Insert: {
          created_at?: string | null
          deleted_at?: string | null
          id: string
          messages?: Json | null
          metadata?: string | null
          session_id: string
          title?: string | null
          updated_at?: string | null
          user_id: string
        }
        Update: {
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          messages?: Json | null
          metadata?: string | null
          session_id?: string
          title?: string | null
          updated_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_claude_sessions_user"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      competition_groups: {
        Row: {
          competition_id: string | null
          graph: string | null
          group_id: string | null
          id: string
          report: string | null
        }
        Insert: {
          competition_id?: string | null
          graph?: string | null
          group_id?: string | null
          id: string
          report?: string | null
        }
        Update: {
          competition_id?: string | null
          graph?: string | null
          group_id?: string | null
          id?: string
          report?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_competition_groups_group"
            columns: ["group_id"]
            isOneToOne: false
            referencedRelation: "groups"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_competitions_competition_groups"
            columns: ["competition_id"]
            isOneToOne: false
            referencedRelation: "competitions"
            referencedColumns: ["id"]
          },
        ]
      }
      competitions: {
        Row: {
          active: boolean | null
          created_at: string | null
          deleted_at: string | null
          end: string | null
          graph: string | null
          id: string
          name: string | null
          start: string | null
          updated_at: string | null
        }
        Insert: {
          active?: boolean | null
          created_at?: string | null
          deleted_at?: string | null
          end?: string | null
          graph?: string | null
          id: string
          name?: string | null
          start?: string | null
          updated_at?: string | null
        }
        Update: {
          active?: boolean | null
          created_at?: string | null
          deleted_at?: string | null
          end?: string | null
          graph?: string | null
          id?: string
          name?: string | null
          start?: string | null
          updated_at?: string | null
        }
        Relationships: []
      }
      content_tags: {
        Row: {
          content_id: string
          tag_id: string
        }
        Insert: {
          content_id: string
          tag_id: string
        }
        Update: {
          content_id?: string
          tag_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_content_tags_content"
            columns: ["content_id"]
            isOneToOne: false
            referencedRelation: "contents"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_content_tags_tag"
            columns: ["tag_id"]
            isOneToOne: false
            referencedRelation: "tags"
            referencedColumns: ["id"]
          },
        ]
      }
      contents: {
        Row: {
          created_at: string | null
          data: string | null
          deleted_at: string | null
          file_size: number | null
          group_id: string
          id: string
          media_url: string | null
          metadata: string | null
          mime_type: string | null
          parent_content_id: string | null
          reply_count: number | null
          type: string
          updated_at: string | null
          user_id: string
        }
        Insert: {
          created_at?: string | null
          data?: string | null
          deleted_at?: string | null
          file_size?: number | null
          group_id: string
          id: string
          media_url?: string | null
          metadata?: string | null
          mime_type?: string | null
          parent_content_id?: string | null
          reply_count?: number | null
          type: string
          updated_at?: string | null
          user_id: string
        }
        Update: {
          created_at?: string | null
          data?: string | null
          deleted_at?: string | null
          file_size?: number | null
          group_id?: string
          id?: string
          media_url?: string | null
          metadata?: string | null
          mime_type?: string | null
          parent_content_id?: string | null
          reply_count?: number | null
          type?: string
          updated_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_contents_group"
            columns: ["group_id"]
            isOneToOne: false
            referencedRelation: "groups"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_contents_thread_replies"
            columns: ["parent_content_id"]
            isOneToOne: false
            referencedRelation: "contents"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_contents_user"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      directions: {
        Row: {
          created_at: string | null
          deleted_at: string | null
          end_time: number | null
          id: string
          recipe_id: string | null
          start_time: number | null
          text: string | null
          updated_at: string | null
        }
        Insert: {
          created_at?: string | null
          deleted_at?: string | null
          end_time?: number | null
          id: string
          recipe_id?: string | null
          start_time?: number | null
          text?: string | null
          updated_at?: string | null
        }
        Update: {
          created_at?: string | null
          deleted_at?: string | null
          end_time?: number | null
          id?: string
          recipe_id?: string | null
          start_time?: number | null
          text?: string | null
          updated_at?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_recipes_directions"
            columns: ["recipe_id"]
            isOneToOne: false
            referencedRelation: "recipes"
            referencedColumns: ["id"]
          },
        ]
      }
      equipment: {
        Row: {
          comment: string | null
          created_at: string | null
          deleted_at: string | null
          id: string
          name: string | null
          recipe_id: string | null
          updated_at: string | null
        }
        Insert: {
          comment?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id: string
          name?: string | null
          recipe_id?: string | null
          updated_at?: string | null
        }
        Update: {
          comment?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          name?: string | null
          recipe_id?: string | null
          updated_at?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_recipes_equipment"
            columns: ["recipe_id"]
            isOneToOne: false
            referencedRelation: "recipes"
            referencedColumns: ["id"]
          },
        ]
      }
      food_names: {
        Row: {
          fdc_id: number | null
          name: string | null
        }
        Insert: {
          fdc_id?: number | null
          name?: string | null
        }
        Update: {
          fdc_id?: number | null
          name?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_food_names_food"
            columns: ["fdc_id"]
            isOneToOne: false
            referencedRelation: "foods"
            referencedColumns: ["fdc_id"]
          },
        ]
      }
      foods: {
        Row: {
          created_at: string | null
          description: string | null
          fdc_id: number
          raw: Json | null
        }
        Insert: {
          created_at?: string | null
          description?: string | null
          fdc_id?: number
          raw?: Json | null
        }
        Update: {
          created_at?: string | null
          description?: string | null
          fdc_id?: number
          raw?: Json | null
        }
        Relationships: []
      }
      group_memberships: {
        Row: {
          created_at: string | null
          group_id: string | null
          id: string
          role: string | null
          user_id: string | null
        }
        Insert: {
          created_at?: string | null
          group_id?: string | null
          id: string
          role?: string | null
          user_id?: string | null
        }
        Update: {
          created_at?: string | null
          group_id?: string | null
          id?: string
          role?: string | null
          user_id?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_groups_members"
            columns: ["group_id"]
            isOneToOne: false
            referencedRelation: "groups"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_users_group_memberships"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      groups: {
        Row: {
          created_at: string | null
          id: string
          join_code: string | null
          name: string
        }
        Insert: {
          created_at?: string | null
          id: string
          join_code?: string | null
          name: string
        }
        Update: {
          created_at?: string | null
          id?: string
          join_code?: string | null
          name?: string
        }
        Relationships: []
      }
      identities: {
        Row: {
          created_at: string | null
          id: string
          provider: string | null
          user_id: string | null
        }
        Insert: {
          created_at?: string | null
          id: string
          provider?: string | null
          user_id?: string | null
        }
        Update: {
          created_at?: string | null
          id?: string
          provider?: string | null
          user_id?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_users_identities"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      ingredients: {
        Row: {
          amount: string | null
          comment: string | null
          created_at: string | null
          deleted_at: string | null
          id: string
          name: string | null
          recipe_id: string | null
          unit: string | null
          updated_at: string | null
        }
        Insert: {
          amount?: string | null
          comment?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id: string
          name?: string | null
          recipe_id?: string | null
          unit?: string | null
          updated_at?: string | null
        }
        Update: {
          amount?: string | null
          comment?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          name?: string | null
          recipe_id?: string | null
          unit?: string | null
          updated_at?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_recipes_ingredients"
            columns: ["recipe_id"]
            isOneToOne: false
            referencedRelation: "recipes"
            referencedColumns: ["id"]
          },
        ]
      }
      pages: {
        Row: {
          article: string | null
          created_at: number | null
          deleted_at: string | null
          group_id: string | null
          hit_count: number | null
          html: string | null
          id: string
          title: string | null
          updated_at: string | null
          url: string | null
        }
        Insert: {
          article?: string | null
          created_at?: number | null
          deleted_at?: string | null
          group_id?: string | null
          hit_count?: number | null
          html?: string | null
          id: string
          title?: string | null
          updated_at?: string | null
          url?: string | null
        }
        Update: {
          article?: string | null
          created_at?: number | null
          deleted_at?: string | null
          group_id?: string | null
          hit_count?: number | null
          html?: string | null
          id?: string
          title?: string | null
          updated_at?: string | null
          url?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_groups_pages"
            columns: ["group_id"]
            isOneToOne: false
            referencedRelation: "groups"
            referencedColumns: ["id"]
          },
        ]
      }
      photo_likes: {
        Row: {
          created_at: string
          id: string
          photo_id: string
          user_id: string
        }
        Insert: {
          created_at?: string
          id?: string
          photo_id: string
          user_id: string
        }
        Update: {
          created_at?: string
          id?: string
          photo_id?: string
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "photo_likes_photo_id_fkey"
            columns: ["photo_id"]
            isOneToOne: false
            referencedRelation: "bread_photos"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "photo_likes_user_id_profiles_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      profiles: {
        Row: {
          avatar_url: string | null
          bio: string | null
          created_at: string
          full_name: string | null
          id: string
          location: string | null
          updated_at: string
          username: string | null
        }
        Insert: {
          avatar_url?: string | null
          bio?: string | null
          created_at?: string
          full_name?: string | null
          id: string
          location?: string | null
          updated_at?: string
          username?: string | null
        }
        Update: {
          avatar_url?: string | null
          bio?: string | null
          created_at?: string
          full_name?: string | null
          id?: string
          location?: string | null
          updated_at?: string
          username?: string | null
        }
        Relationships: []
      }
      prompt_contexts: {
        Row: {
          context_id: string | null
          created_at: string | null
          deleted_at: string | null
          id: string
          request: string | null
          response: string | null
          updated_at: string | null
        }
        Insert: {
          context_id?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id: string
          request?: string | null
          response?: string | null
          updated_at?: string | null
        }
        Update: {
          context_id?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          request?: string | null
          response?: string | null
          updated_at?: string | null
        }
        Relationships: []
      }
      prompt_runs: {
        Row: {
          created_at: string | null
          deleted_at: string | null
          id: string
          input: string | null
          output: string | null
          prompt_id: string | null
          updated_at: string | null
        }
        Insert: {
          created_at?: string | null
          deleted_at?: string | null
          id: string
          input?: string | null
          output?: string | null
          prompt_id?: string | null
          updated_at?: string | null
        }
        Update: {
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          input?: string | null
          output?: string | null
          prompt_id?: string | null
          updated_at?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_prompts_runs"
            columns: ["prompt_id"]
            isOneToOne: false
            referencedRelation: "prompts"
            referencedColumns: ["id"]
          },
        ]
      }
      prompts: {
        Row: {
          content: string | null
          created_at: string | null
          deleted_at: string | null
          id: string
          parent_id: string | null
          title: string | null
          updated_at: string | null
          user_id: string | null
        }
        Insert: {
          content?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id: string
          parent_id?: string | null
          title?: string | null
          updated_at?: string | null
          user_id?: string | null
        }
        Update: {
          content?: string | null
          created_at?: string | null
          deleted_at?: string | null
          id?: string
          parent_id?: string | null
          title?: string | null
          updated_at?: string | null
          user_id?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "fk_prompts_forks"
            columns: ["parent_id"]
            isOneToOne: false
            referencedRelation: "prompts"
            referencedColumns: ["id"]
          },
        ]
      }
      recipe_tags: {
        Row: {
          recipe_id: string
          tag_id: string
        }
        Insert: {
          recipe_id: string
          tag_id: string
        }
        Update: {
          recipe_id?: string
          tag_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_recipe_tags_recipe"
            columns: ["recipe_id"]
            isOneToOne: false
            referencedRelation: "recipes"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "fk_recipe_tags_tag"
            columns: ["tag_id"]
            isOneToOne: false
            referencedRelation: "tags"
            referencedColumns: ["id"]
          },
        ]
      }
      recipes: {
        Row: {
          created_at: string | null
          deleted_at: string | null
          domain: string | null
          id: string
          name: string | null
          transcript: string | null
          updated_at: string | null
          url: string | null
        }
        Insert: {
          created_at?: string | null
          deleted_at?: string | null
          domain?: string | null
          id: string
          name?: string | null
          transcript?: string | null
          updated_at?: string | null
          url?: string | null
        }
        Update: {
          created_at?: string | null
          deleted_at?: string | null
          domain?: string | null
          id?: string
          name?: string | null
          transcript?: string | null
          updated_at?: string | null
          url?: string | null
        }
        Relationships: []
      }
      tags: {
        Row: {
          color: string | null
          created_at: string | null
          deleted_at: string | null
          description: string | null
          id: string
          name: string
          updated_at: string | null
          user_id: string
        }
        Insert: {
          color?: string | null
          created_at?: string | null
          deleted_at?: string | null
          description?: string | null
          id: string
          name: string
          updated_at?: string | null
          user_id: string
        }
        Update: {
          color?: string | null
          created_at?: string | null
          deleted_at?: string | null
          description?: string | null
          id?: string
          name?: string
          updated_at?: string | null
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "fk_tags_user"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "users"
            referencedColumns: ["id"]
          },
        ]
      }
      user_baking_history: {
        Row: {
          bread_id: string
          bread_name: string
          completed_steps: string[] | null
          created_at: string
          duration_minutes: number | null
          finish_time: string
          id: string
          notes: string | null
          photo_ids: string[] | null
          rating: number | null
          start_time: string
          total_steps: number
          user_id: string
        }
        Insert: {
          bread_id: string
          bread_name: string
          completed_steps?: string[] | null
          created_at?: string
          duration_minutes?: number | null
          finish_time: string
          id?: string
          notes?: string | null
          photo_ids?: string[] | null
          rating?: number | null
          start_time: string
          total_steps?: number
          user_id: string
        }
        Update: {
          bread_id?: string
          bread_name?: string
          completed_steps?: string[] | null
          created_at?: string
          duration_minutes?: number | null
          finish_time?: string
          id?: string
          notes?: string | null
          photo_ids?: string[] | null
          rating?: number | null
          start_time?: string
          total_steps?: number
          user_id?: string
        }
        Relationships: [
          {
            foreignKeyName: "user_baking_history_bread_id_fkey"
            columns: ["bread_id"]
            isOneToOne: false
            referencedRelation: "user_recipes"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "user_baking_history_user_id_profiles_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      user_recipes: {
        Row: {
          created_at: string
          description: string | null
          difficulty: string | null
          id: string
          image_url: string | null
          ingredients: Json | null
          is_public: boolean | null
          name: string
          prep_time: number | null
          steps: Json | null
          tags: string[] | null
          total_time: number | null
          updated_at: string
          user_id: string
        }
        Insert: {
          created_at?: string
          description?: string | null
          difficulty?: string | null
          id?: string
          image_url?: string | null
          ingredients?: Json | null
          is_public?: boolean | null
          name: string
          prep_time?: number | null
          steps?: Json | null
          tags?: string[] | null
          total_time?: number | null
          updated_at?: string
          user_id: string
        }
        Update: {
          created_at?: string
          description?: string | null
          difficulty?: string | null
          id?: string
          image_url?: string | null
          ingredients?: Json | null
          is_public?: boolean | null
          name?: string
          prep_time?: number | null
          steps?: Json | null
          tags?: string[] | null
          total_time?: number | null
          updated_at?: string
          user_id?: string
        }
        Relationships: []
      }
      users: {
        Row: {
          created_at: string | null
          id: string
          password: string | null
          username: string | null
        }
        Insert: {
          created_at?: string | null
          id: string
          password?: string | null
          username?: string | null
        }
        Update: {
          created_at?: string | null
          id?: string
          password?: string | null
          username?: string | null
        }
        Relationships: []
      }
    }
    Views: {
      ai_recipe_generation_analytics: {
        Row: {
          ai_model: string | null
          avg_api_time_ms: number | null
          avg_processing_time_ms: number | null
          date: string | null
          failed_generations: number | null
          generation_type: string | null
          successful_generations: number | null
          total_generations: number | null
          total_images_processed: number | null
          unique_users: number | null
        }
        Relationships: []
      }
      community_bread_progress: {
        Row: {
          avatar_url: string | null
          bread_id: string | null
          bread_name: string | null
          completed_steps: string[] | null
          current_step: number | null
          expected_finish_time: string | null
          id: string | null
          last_updated: string | null
          location: string | null
          recipe_difficulty: string | null
          recipe_name: string | null
          start_time: string | null
          user_id: string | null
          username: string | null
        }
        Relationships: [
          {
            foreignKeyName: "bread_progress_bread_id_fkey"
            columns: ["bread_id"]
            isOneToOne: false
            referencedRelation: "user_recipes"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "bread_progress_user_id_profiles_fkey"
            columns: ["user_id"]
            isOneToOne: false
            referencedRelation: "profiles"
            referencedColumns: ["id"]
          },
        ]
      }
      user_complete_history: {
        Row: {
          bread_id: string | null
          bread_name: string | null
          completed_steps: string[] | null
          current_step: number | null
          finish_time: string | null
          id: string | null
          is_completed: boolean | null
          last_updated: string | null
          source_table: string | null
          start_time: string | null
          user_id: string | null
        }
        Relationships: []
      }
    }
    Functions: {
      [_ in never]: never
    }
    Enums: {
      [_ in never]: never
    }
    CompositeTypes: {
      [_ in never]: never
    }
  }
}

type DefaultSchema = Database[Extract<keyof Database, "public">]

export type Tables<
  DefaultSchemaTableNameOrOptions extends
    | keyof (DefaultSchema["Tables"] & DefaultSchema["Views"])
    | { schema: keyof Database },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof Database
  }
    ? keyof (Database[DefaultSchemaTableNameOrOptions["schema"]]["Tables"] &
        Database[DefaultSchemaTableNameOrOptions["schema"]]["Views"])
    : never = never,
> = DefaultSchemaTableNameOrOptions extends { schema: keyof Database }
  ? (Database[DefaultSchemaTableNameOrOptions["schema"]]["Tables"] &
      Database[DefaultSchemaTableNameOrOptions["schema"]]["Views"])[TableName] extends {
      Row: infer R
    }
    ? R
    : never
  : DefaultSchemaTableNameOrOptions extends keyof (DefaultSchema["Tables"] &
        DefaultSchema["Views"])
    ? (DefaultSchema["Tables"] &
        DefaultSchema["Views"])[DefaultSchemaTableNameOrOptions] extends {
        Row: infer R
      }
      ? R
      : never
    : never

export type TablesInsert<
  DefaultSchemaTableNameOrOptions extends
    | keyof DefaultSchema["Tables"]
    | { schema: keyof Database },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof Database
  }
    ? keyof Database[DefaultSchemaTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = DefaultSchemaTableNameOrOptions extends { schema: keyof Database }
  ? Database[DefaultSchemaTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Insert: infer I
    }
    ? I
    : never
  : DefaultSchemaTableNameOrOptions extends keyof DefaultSchema["Tables"]
    ? DefaultSchema["Tables"][DefaultSchemaTableNameOrOptions] extends {
        Insert: infer I
      }
      ? I
      : never
    : never

export type TablesUpdate<
  DefaultSchemaTableNameOrOptions extends
    | keyof DefaultSchema["Tables"]
    | { schema: keyof Database },
  TableName extends DefaultSchemaTableNameOrOptions extends {
    schema: keyof Database
  }
    ? keyof Database[DefaultSchemaTableNameOrOptions["schema"]]["Tables"]
    : never = never,
> = DefaultSchemaTableNameOrOptions extends { schema: keyof Database }
  ? Database[DefaultSchemaTableNameOrOptions["schema"]]["Tables"][TableName] extends {
      Update: infer U
    }
    ? U
    : never
  : DefaultSchemaTableNameOrOptions extends keyof DefaultSchema["Tables"]
    ? DefaultSchema["Tables"][DefaultSchemaTableNameOrOptions] extends {
        Update: infer U
      }
      ? U
      : never
    : never

export type Enums<
  DefaultSchemaEnumNameOrOptions extends
    | keyof DefaultSchema["Enums"]
    | { schema: keyof Database },
  EnumName extends DefaultSchemaEnumNameOrOptions extends {
    schema: keyof Database
  }
    ? keyof Database[DefaultSchemaEnumNameOrOptions["schema"]]["Enums"]
    : never = never,
> = DefaultSchemaEnumNameOrOptions extends { schema: keyof Database }
  ? Database[DefaultSchemaEnumNameOrOptions["schema"]]["Enums"][EnumName]
  : DefaultSchemaEnumNameOrOptions extends keyof DefaultSchema["Enums"]
    ? DefaultSchema["Enums"][DefaultSchemaEnumNameOrOptions]
    : never

export type CompositeTypes<
  PublicCompositeTypeNameOrOptions extends
    | keyof DefaultSchema["CompositeTypes"]
    | { schema: keyof Database },
  CompositeTypeName extends PublicCompositeTypeNameOrOptions extends {
    schema: keyof Database
  }
    ? keyof Database[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"]
    : never = never,
> = PublicCompositeTypeNameOrOptions extends { schema: keyof Database }
  ? Database[PublicCompositeTypeNameOrOptions["schema"]]["CompositeTypes"][CompositeTypeName]
  : PublicCompositeTypeNameOrOptions extends keyof DefaultSchema["CompositeTypes"]
    ? DefaultSchema["CompositeTypes"][PublicCompositeTypeNameOrOptions]
    : never

export const Constants = {
  public: {
    Enums: {},
  },
} as const
