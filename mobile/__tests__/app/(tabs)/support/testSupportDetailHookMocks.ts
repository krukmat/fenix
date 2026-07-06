export const mockUseCase = jest.fn();
export const mockUseEntityTimeline = jest.fn();
export const mockUseCaseActivities = jest.fn();
export const mockUseCaseNotes = jest.fn();

export function mockSupportDetailUseCRMModule() {
  return {
    useCase: (...args: unknown[]) => mockUseCase(...args),
    useEntityTimeline: (...args: unknown[]) => mockUseEntityTimeline(...args),
    useCaseActivities: (...args: unknown[]) => mockUseCaseActivities(...args),
    useCaseNotes: (...args: unknown[]) => mockUseCaseNotes(...args),
  };
}

export function seedEmptySupportDetailQueries() {
  mockUseEntityTimeline.mockReturnValue({ data: { data: [] }, isLoading: false });
  mockUseCaseActivities.mockReturnValue({ data: { data: [] }, isLoading: false });
  mockUseCaseNotes.mockReturnValue({ data: { data: [] }, isLoading: false });
}
