---
name: react-native
version: 1.0.0
description: React Native cross-platform mobile development best practices
---

## React Native Development Instructions

When building React Native applications, follow these guidelines:

1. **Component Architecture**: Use a presentational/container pattern. Keep business logic in custom hooks, not in components.
2. **State Management**: Use React Context + useReducer for medium apps, Zustand or Jotai for complex state. Avoid Redux unless the app is very large.
3. **Navigation**: Use React Navigation v6+. Define typed route parameters with TypeScript generics for type-safe navigation.
4. **Performance**: Use FlatList (not ScrollView) for long lists. Apply getItemLayout and keyExtractor for optimal rendering. Use React.memo and useMemo strategically.
5. **Native Modules**: Write native modules (Kotlin/Swift) for platform-specific features. Expose clean TypeScript interfaces to JS.
6. **Offline First**: Implement offline-first with MMKV for fast storage and React Query for sync. Queue writes and reconcile on connectivity restore.
7. **Testing**: Write unit tests with Jest + React Native Testing Library. Use Detox or Maestro for E2E tests.
8. **CI/CD**: Use EAS Build for iOS/Android builds. Implement code push (Expo Updates) for hot fixes without store review.
9. **Accessibility**: Set accessible={true} on interactive elements. Provide accessibilityLabel and accessibilityHint. Test with screen readers.
10. **Security**: Store sensitive data in Keychain (iOS) / EncryptedSharedPreferences (Android). Never store tokens in AsyncStorage. Use SSL pinning for API calls.
11. **Analytics**: Instrument with a provider-agnostic analytics layer (Segment or custom). Track screen views, user actions, and errors.
12. **Internationalization**: Use i18next for multi-language support. Store user language preference and fall back to device locale.
