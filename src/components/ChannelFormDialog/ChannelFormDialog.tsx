import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Plus, Edit } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { components } from '../../lib/api/v1';

type Channel = components["schemas"]["Channel"];

const channelSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  guide_id: z.string().min(1, 'Guide ID is required'),
  logo: z.string().url('Must be a valid URL').or(z.literal('')),
  group_title: z.string().min(1, 'Group is required'),
});

type ChannelFormData = z.infer<typeof channelSchema>;

interface ChannelFormDialogProps {
  mode: 'create' | 'edit';
  channel?: Channel;
  onSubmit: (data: ChannelFormData) => Promise<void>;
}

export default function ChannelFormDialog({
  mode,
  channel,
  onSubmit,
}: ChannelFormDialogProps) {
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<ChannelFormData>({
    resolver: zodResolver(channelSchema),
    defaultValues: channel
      ? {
          title: channel.title,
          guide_id: channel.guide_id,
          logo: channel.logo || '',
          group_title: channel.group_title,
        }
      : {
          title: '',
          guide_id: '',
          logo: '',
          group_title: '',
        },
  });

  const handleFormSubmit = async (data: ChannelFormData) => {
    setIsSubmitting(true);
    try {
      await onSubmit(data);
      setOpen(false);
      reset();
    } catch (error) {
      console.error('Failed to save channel:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      reset();
    }
  };

  return (
    <>
      <Button
        variant={mode === 'create' ? 'default' : 'outline'}
        size="sm"
        onClick={() => setOpen(true)}
      >
        {mode === 'create' ? (
          <>
            <Plus className="h-4 w-4" />
            Add Channel
          </>
        ) : (
          <>
            <Edit className="h-4 w-4" />
            Edit
          </>
        )}
      </Button>

      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-[500px]">
          <form onSubmit={handleSubmit(handleFormSubmit)}>
            <DialogHeader>
              <DialogTitle>
                {mode === 'create' ? 'Add New Channel' : 'Edit Channel'}
              </DialogTitle>
              <DialogDescription>
                {mode === 'create'
                  ? 'Create a new channel. You can add streams later.'
                  : 'Update the channel information.'}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="title">Title *</Label>
                <Input
                  id="title"
                  {...register('title')}
                  placeholder="Channel Name"
                />
                {errors.title && (
                  <p className="text-sm text-red-600">{errors.title.message}</p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="guide_id">Guide ID *</Label>
                <Input
                  id="guide_id"
                  {...register('guide_id')}
                  placeholder="EPG ID"
                />
                {errors.guide_id && (
                  <p className="text-sm text-red-600">{errors.guide_id.message}</p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="logo">Logo URL</Label>
                <Input
                  id="logo"
                  {...register('logo')}
                  placeholder="https://example.com/logo.png"
                />
                {errors.logo && (
                  <p className="text-sm text-red-600">{errors.logo.message}</p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="group_title">Group/Category *</Label>
                <Input
                  id="group_title"
                  {...register('group_title')}
                  placeholder="Sports, Movies, etc."
                />
                {errors.group_title && (
                  <p className="text-sm text-red-600">
                    {errors.group_title.message}
                  </p>
                )}
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting
                  ? 'Saving...'
                  : mode === 'create'
                  ? 'Create Channel'
                  : 'Save Changes'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
