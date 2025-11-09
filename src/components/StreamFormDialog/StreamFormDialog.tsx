import { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
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
import EPGChannelCombobox from '../EPGChannelCombobox/EPGChannelCombobox';

type Stream = components["schemas"]["Stream"];

const streamSchema = z.object({
  guide_id: z.string().min(1, 'Guide ID is required'),
  acestream_id: z.string().min(1, 'Acestream ID is required'),
  quality: z.string(),
  tags: z.string(), // Comma-separated, will be split
  network_caching: z.number().min(0, 'Must be >= 0'),
});

type StreamFormData = z.infer<typeof streamSchema>;

interface StreamFormDialogProps {
  mode: 'create' | 'edit';
  stream?: Stream;
  guideId?: string; // Pre-filled guide_id when adding stream to existing channel
  onSubmit: (data: Omit<Stream, 'id'>) => Promise<void>;
}

export default function StreamFormDialog({
  mode,
  stream,
  guideId,
  onSubmit,
}: StreamFormDialogProps) {
  const [open, setOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
    reset,
  } = useForm<StreamFormData>({
    resolver: zodResolver(streamSchema),
    defaultValues: stream
      ? {
          guide_id: stream.guide_id,
          acestream_id: stream.acestream_id,
          quality: stream.quality || '',
          tags: stream.tags?.join(', ') || '',
          network_caching: stream.network_caching,
        }
      : {
          guide_id: guideId || '',
          acestream_id: '',
          quality: '',
          tags: '',
          network_caching: 10000,
        },
  });

  const handleFormSubmit = async (data: StreamFormData) => {
    setIsSubmitting(true);
    try {
      await onSubmit({
        guide_id: data.guide_id,
        acestream_id: data.acestream_id,
        quality: data.quality,
        tags: data.tags.split(',').map(t => t.trim()).filter(t => t),
        network_caching: data.network_caching,
      });
      setOpen(false);
      reset();
    } catch (error) {
      console.error('Failed to save stream:', error);
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
        variant={mode === 'create' ? 'secondary' : 'ghost'}
        size="sm"
        onClick={() => setOpen(true)}
      >
        {mode === 'create' ? (
          <>
            <Plus className="h-4 w-4" />
            Add Stream
          </>
        ) : (
          <>
            <Edit className="h-3 w-3" />
          </>
        )}
      </Button>

      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-[500px]">
          <form onSubmit={handleSubmit(handleFormSubmit)}>
            <DialogHeader>
              <DialogTitle>
                {mode === 'create' ? 'Add New Stream' : 'Edit Stream'}
              </DialogTitle>
              <DialogDescription>
                {mode === 'create'
                  ? 'Add a new stream. Select channel from EPG.'
                  : 'Update the stream information.'}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              {/* Guide ID field - show combobox if not pre-filled, otherwise show readonly */}
              {guideId ? (
                <div className="grid gap-2">
                  <Label htmlFor="guide_id">Guide ID (Channel)</Label>
                  <Input
                    id="guide_id"
                    value={guideId}
                    disabled
                    className="bg-gray-100 dark:bg-gray-700"
                  />
                </div>
              ) : (
                <div className="grid gap-2">
                  <Label htmlFor="guide_id">Guide ID (Channel) *</Label>
                  <Controller
                    name="guide_id"
                    control={control}
                    render={({ field }) => (
                      <EPGChannelCombobox
                        value={field.value}
                        onChange={(guideId) => {
                          field.onChange(guideId);
                        }}
                        error={errors.guide_id?.message}
                      />
                    )}
                  />
                  {errors.guide_id && (
                    <p className="text-sm text-red-600 dark:text-red-400">
                      {errors.guide_id.message}
                    </p>
                  )}
                </div>
              )}

              <div className="grid gap-2">
                <Label htmlFor="acestream_id">Acestream ID *</Label>
                <Input
                  id="acestream_id"
                  {...register('acestream_id')}
                  placeholder="40-character hex string"
                  className="font-mono text-sm"
                />
                {errors.acestream_id && (
                  <p className="text-sm text-red-600 dark:text-red-400">
                    {errors.acestream_id.message}
                  </p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="quality">Quality</Label>
                <Input
                  id="quality"
                  {...register('quality')}
                  placeholder="SD, HD, FHD, 4K"
                />
                {errors.quality && (
                  <p className="text-sm text-red-600 dark:text-red-400">{errors.quality.message}</p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="tags">Tags (comma-separated)</Label>
                <Input
                  id="tags"
                  {...register('tags')}
                  placeholder="Spanish, English, etc."
                />
                {errors.tags && (
                  <p className="text-sm text-red-600 dark:text-red-400">{errors.tags.message}</p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="network_caching">
                  Network Caching (ms) *
                </Label>
                <Input
                  id="network_caching"
                  type="number"
                  {...register('network_caching', { valueAsNumber: true })}
                  placeholder="10000"
                />
                {errors.network_caching && (
                  <p className="text-sm text-red-600 dark:text-red-400">
                    {errors.network_caching.message}
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
                  ? 'Add Stream'
                  : 'Save Changes'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
